package pipeline

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// getRepoName extracts the repository name from a Git URL.
func getRepoName() (string, error) {
	// get the component name from the git repo name by running git
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git remote origin: %w", err)
	}
	gitURL := string(output)
	repoName, err := getRepoNameFromGitURL(gitURL)
	if err != nil {
		return "", fmt.Errorf("failed to get repo name from git URL: %w", err)
	}
	return repoName, nil
}

// getRepoNameFromGitURL extracts the repository name from a Git URL.
func getRepoNameFromGitURL(gitURL string) (string, error) {
	regex := regexp.MustCompile(`[:/]([^/:]+/[^/]+)\.git$`)
	matches := regex.FindStringSubmatch(gitURL)

	if len(matches) == 0 {
		return "", fmt.Errorf("no repository name found in URL: %s", gitURL)
	}

	repoParts := regexp.MustCompile(`/`).Split(matches[1], -1)
	repoName := repoParts[len(repoParts)-1]

	return repoName, nil
}

// getGitSHA gets the current Git SHA.
func getGitSHA() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git SHA: %w", err)
	}
	return string(output), nil
}

// LoadMetadata gets the metadata from the environment.
func LoadMetadata() (*Metadata, error) {
	repoName, err := getRepoName()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo name: %w", err)
	}
	gitSha, err := getGitSHA()
	if err != nil {
		return nil, fmt.Errorf("failed to get git SHA: %w", err)
	}
	return &Metadata{
		Component: repoName,
		GitSHA:    gitSha,
		BuildID:   "TODO",
	}, nil
}

// LoadPipeline loads and parses the pipeline YAML file.
func LoadPipeline(filePath string) (*Pipeline, error) {
	var pipeline Pipeline
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	fileContents, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(fileContents, &pipeline)
	if err != nil {
		return nil, err
	}
	return &pipeline, nil
}

// Run the pipeline, streaming events to the provided channel.
func (p *Pipeline) Run(metadata *Metadata) chan Event {
	events := make(chan Event, 32)
	go func() {
		defer close(events)
		events <- PipelineStartEvent{BaseEvent{EventTime: time.Now()}}
		env := os.Environ()
		for k, v := range p.GlobalEnv {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		env = append(env, fmt.Sprintf("COMPONENT=%s", metadata.Component))
		env = append(env, fmt.Sprintf("GIT_SHA=%s", metadata.GitSHA))
		env = append(env, fmt.Sprintf("BUILD_ID=%s", metadata.BuildID))
		for _, step := range p.Steps {
			events <- StepStartEvent{BaseEvent{EventTime: time.Now()}, step.Name}
			err := p.runStep(env, step, events)
			if err != nil {
				events <- StepFailureEvent{BaseEvent{EventTime: time.Now()}, fmt.Sprintf("step '%s' failed", step.Name), err.Error()}
				events <- PipelineFailureEvent{BaseEvent{EventTime: time.Now()}, fmt.Sprintf("step '%s' failed", step.Name)}
				return
			}
			events <- StepSuccessEvent{BaseEvent{EventTime: time.Now()}, step.Name}
		}
		events <- PipelineSuccessEvent{BaseEvent{EventTime: time.Now()}}
	}()
	return events
}

func (p *Pipeline) runStep(env []string, step Step, events chan Event) error {
	cmd := exec.Command(step.Script)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error)
	go func() {
		var remainder []byte
		buffer := make([]byte, 16*1024)
		for {
			n, err := stdout.Read(buffer)
			if err != nil {
				if err == io.EOF {
					break
				}
				done <- err
				return
			}
			chunk := append(remainder, buffer[:n]...)
			i := bytes.LastIndex(chunk, []byte{'\n'})
			if i == -1 {
				remainder = chunk
				continue
			}
			events <- StepOutputEvent{
				BaseEvent{EventTime: time.Now()},
				step.Name,
				string(chunk[:i+1]),
			}
			remainder = chunk[i+1:]
		}
		if len(remainder) > 0 {
			events <- StepOutputEvent{
				BaseEvent{EventTime: time.Now()},
				step.Name,
				string(remainder),
			}
		}
		done <- nil
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	if err := <-done; err != nil {
		return fmt.Errorf("reading command output failed: %w", err)
	}

	return nil
}

// checkUncommittedChanges checks if there are any uncommitted changes in the local repository.
func checkUncommittedChanges() error {
	cmd := exec.Command("git", "status", "--porcelain")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return err
	}
	if out.String() != "" {
		return fmt.Errorf("uncommitted changes present")
	}
	return nil
}

// checkIfLocalIsAheadOfRemote checks if the local branch is ahead of the remote branch.
func checkIfLocalIsAheadOfRemote(remote, branch string) error {
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", fmt.Sprintf("%s/%s...HEAD", remote, branch))
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return err
	}
	counts := strings.Fields(out.String())
	if len(counts) != 2 {
		return fmt.Errorf("unexpected output from rev-list")
	}
	if counts[0] != "0" {
		return fmt.Errorf("local branch is ahead of remote by %s commits", counts[0])
	}
	return nil
}
