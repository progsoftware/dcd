package dcd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// NewPipeline creates a new Pipeline
func NewPipeline() *Pipeline {
	return &Pipeline{}
}

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
func (p *Pipeline) LoadMetadata() error {
	repoName, err := getRepoName()
	if err != nil {
		return fmt.Errorf("failed to get repo name: %w", err)
	}
	gitSha, err := getGitSHA()
	if err != nil {
		return fmt.Errorf("failed to get git SHA: %w", err)
	}
	p.metadata = &Metadata{
		Component: repoName,
		GitSHA:    gitSha,
	}
	return nil
}

// SetMetadata sets the metadata.
func (p *Pipeline) SetMetadata(metadata *Metadata) {
	p.metadata = metadata
}

// LoadPipeline loads and parses the pipeline YAML file.
func (p *Pipeline) LoadPipelineDefinition(filePath string) error {
	var definition PipelineDefinition
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	fileContents, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(fileContents, &definition)
	if err != nil {
		return err
	}
	p.definition = &definition
	return nil
}

// SetDefinition sets the pipeline definition.
func (p *Pipeline) SetDefinition(definition *PipelineDefinition) {
	p.definition = definition
}

// SetBackend sets the backend.
func (p *Pipeline) SetBackend(backend Backend) {
	p.backend = backend
}

// Run the pipeline, streaming events to the provided channel.
func (p *Pipeline) Run() (chan Event, error) {
	if err := checkUncommittedChanges(); err != nil {
		return nil, err
	}
	if err := checkIfLocalIsAheadOfRemote("origin", "main"); err != nil {
		return nil, err
	}
	ctx := context.Background()
	buildID, err := p.backend.GetBuildID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get build ID: %w", err)
	}

	state := &PipelineState{
		Status:  "pending",
		BuildID: buildID,
	}

	if err := p.backend.PutPipeline(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to put pipeline: %w", err)
	}

	events := make(chan Event, 32)
	go func() {
		defer close(events)

		events <- PipelineStartEvent{
			BaseEvent: BaseEvent{EventTime: time.Now()},
			BuildID:   buildID,
		}
		env := os.Environ()
		for k, v := range p.definition.GlobalEnv {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		env = append(env, fmt.Sprintf("COMPONENT=%s", p.metadata.Component))
		env = append(env, fmt.Sprintf("GIT_SHA=%s", p.metadata.GitSHA))
		env = append(env, fmt.Sprintf("BUILD_ID=%d", buildID))
		for _, step := range p.definition.Steps {
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
	return events, nil
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
		return &UncommittedChangesError{}
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
	remoteAhead, err := strconv.Atoi(counts[0])
	if err != nil {
		return fmt.Errorf("failed to parse remote ahead count: %w", err)
	}
	localAhead, err := strconv.Atoi(counts[1])
	if err != nil {
		return fmt.Errorf("failed to parse local ahead count: %w", err)
	}
	if remoteAhead > 0 || localAhead > 0 {
		return &UnsyncedChangesError{remoteAhead, localAhead}
	}
	return nil
}
