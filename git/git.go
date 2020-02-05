package git

import (
	"fmt"
	"os/exec"
	"regexp"

	"github.com/pkg/errors"
)

type VcsType string

const (
	GitHub    VcsType = "GITHUB"
	BitBucket VcsType = "BITBUCKET"
)

func InferOrganizationFromGitRemotes() (VcsType, string, error) {

	// Use some regexes to parse the output of `git remote` to infer what VCS provider
	// is being used by the customer in this directory. The assumtpion here is that
	// the 'origin' remote will be a bitbucket or github project. This matching is
	// a best effort approach, and pull requests are welcome to make it more robust.

	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", errors.Wrap(err, "Error finding the 'origin' git remote")
	}
	output := string(out)

	// SSH
	// git@github.com:circleci/api-service.git
	// git@bitbucket.org:dellelce/makefile_sh.git
	// HTTPS
	// https://github.com/circleci/esxi-api.git
	// https://marcomorain_ci@bitbucket.org/dellelce/makefile_sh.git
	ssh := regexp.MustCompile(`^git@(github\.com|bitbucket\.org):(.*)/`)
	https := regexp.MustCompile(`https://(?:.*@)?(github\.com|bitbucket\.org)/(.*)/`)

	matches := ssh.FindStringSubmatch(output)
	if matches == nil {
		matches = https.FindStringSubmatch(output)
	}

	if matches == nil {
		return "", "", errors.New("Unable to determine VCS information from git 'origin' remote")
	}

	domain := string(matches[1])
	org := string(matches[2])

	switch domain {
	case "github.com":
		return GitHub, org, nil
	case "bitbucket.org":
		return BitBucket, org, nil
	}

	panic(fmt.Sprintf("regexp error: domain=%v org=%v matches=%v", domain, org, matches))
}
