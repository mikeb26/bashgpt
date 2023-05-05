/* Copyright © 2023 Mike Brown. All Rights Reserved.
 *
 * See LICENSE file at the root of this package for license terms
 */
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sashabaranov/go-openai"
)

const (
	CommandName        = "bashgpt"
	KeyFile            = ".openai.key"
	AutocompleteScript = "bashgpt_autocomplete.sh"
	PromptDelim        = "--"
)

const SystemMsg = `You are bash shell autocompletion utility. Users are invoking
you via a terminal by writing a query at the bash prompt and utilizing bash's
autocomplete feature to convert their query into appropriate bash
commands. Please write your responses in strict bash shell syntax without any
additional information. Only respond with a single code block which encapsultes
the user's query. When responding with a code block that includes curl or wget,
please explicitly specify the same user agent as Google Chrome on Windows 10.`

var subCommandTab = map[string]func(args []string) error{
	"config":  configMain,
	"help":    helpMain,
	"sh":      shMain,
	"upgrade": upgradeMain,
	"version": versionMain,
}

func configMain(args []string) error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}
	err = os.MkdirAll(configDir, 0700)
	if err != nil {
		return fmt.Errorf("Could not create config directory %v: %w",
			configDir, err)
	}
	keyPath := path.Join(configDir, KeyFile)
	_, err = os.Stat(keyPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Could not open OpenAI API key file %v: %w", keyPath, err)
	}
	fmt.Printf("Enter your OpenAI API key: ")
	key := ""
	fmt.Scanf("%s", &key)
	key = strings.TrimSpace(key)
	err = ioutil.WriteFile(keyPath, []byte(key), 0600)
	if err != nil {
		return fmt.Errorf("Could not write OpenAI API key file %v: %w", keyPath, err)
	}

	err = checkLatestAutocompleteScript()
	if err != nil {
		return fmt.Errorf("Could not write autocomplete script: %w", keyPath, err)
	}

	fmt.Printf("Add the following to your .bashrc or equivalent:\n  if [ -f ~/.config/%v/%v ]; then\n      . ~/.config/%v/%v\n  fi\n",
		CommandName, AutocompleteScript, CommandName, AutocompleteScript)

	return nil
}

func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("Could not find user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", CommandName), nil
}

func getKeyPath() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, KeyFile), nil
}

func loadKey() (string, error) {
	keyPath, err := getKeyPath()
	if err != nil {
		return "", fmt.Errorf("Could not load OpenAI API key: %w", err)
	}
	data, err := ioutil.ReadFile(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("Could not load OpenAI API key: "+
				"run `%v config` to configure", CommandName)
		}
		return "", fmt.Errorf("Could not load OpenAI API key: %w", err)
	}
	return string(data), nil
}

//go:embed help.txt
var helpText string

func helpMain(args []string) error {
	fmt.Printf(helpText)

	return nil
}

//go:embed version.txt
var versionText string

const DevVersionText = "v0.devbuild"

func versionMain(args []string) error {
	fmt.Printf("%v-%v\n", CommandName, versionText)

	return nil
}

func upgradeMain(args []string) error {
	if versionText == DevVersionText {
		fmt.Fprintf(os.Stderr, "Skipping %v upgrade on development version\n", CommandName)
		return nil
	}
	latestVer, err := getLatestVersion()
	if err != nil {
		return err
	}
	if latestVer == versionText {
		fmt.Printf("%v %v is already the latest version\n", CommandName, versionText)
		return nil
	}

	fmt.Printf("A new version of %v is available (%v). Upgrade? (Y/N) [Y]: ", CommandName,
		latestVer)
	shouldUpgrade := "Y"
	fmt.Scanf("%s", &shouldUpgrade)
	shouldUpgrade = strings.ToUpper(strings.TrimSpace(shouldUpgrade))

	if shouldUpgrade[0] != 'Y' {
		return nil
	}

	fmt.Printf("Upgrading %v from %v to %v...\n", versionText, CommandName,
		latestVer)

	return upgradeViaGithub(latestVer)
}

func getLatestVersion() (string, error) {
	const LatestReleaseUrl = "https://api.github.com/repos/mikeb26/bashgpt/releases/latest"

	client := http.Client{
		Timeout: time.Second * 30,
	}

	resp, err := client.Get(LatestReleaseUrl)
	if err != nil {
		return "", err
	}

	releaseJsonDoc, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var releaseDoc map[string]any
	err = json.Unmarshal(releaseJsonDoc, &releaseDoc)
	if err != nil {
		return "", err
	}

	latestRelease, ok := releaseDoc["tag_name"].(string)
	if !ok {
		return "", fmt.Errorf("Could not parse %v", LatestReleaseUrl)
	}

	return latestRelease, nil
}

func upgradeViaGithub(latestVer string) error {
	const LatestDownloadFmt = "https://github.com/mikeb26/bashgpt/releases/download/%v/bashgpt"

	client := http.Client{
		Timeout: time.Second * 30,
	}

	resp, err := client.Get(fmt.Sprintf(LatestDownloadFmt, latestVer))
	if err != nil {
		return fmt.Errorf("Failed to download version %v: %w", versionText, err)

	}

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("%v-*", CommandName))
	if err != nil {
		return fmt.Errorf("Failed to create temp file: %w", err)
	}
	binaryContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failed to download version %v: %w", versionText, err)
	}
	_, err = tmpFile.Write(binaryContent)
	if err != nil {
		return fmt.Errorf("Failed to download version %v: %w", versionText, err)
	}
	err = tmpFile.Chmod(0755)
	if err != nil {
		return fmt.Errorf("Failed to download version %v: %w", versionText, err)
	}
	err = tmpFile.Close()
	if err != nil {
		return fmt.Errorf("Failed to download version %v: %w", versionText, err)
	}
	myBinaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("Could not determine path to %v: %w", CommandName, err)
	}
	myBinaryPath, err = filepath.EvalSymlinks(myBinaryPath)
	if err != nil {
		return fmt.Errorf("Could not determine path to %v: %w", CommandName, err)
	}

	myBinaryPathBak := myBinaryPath + ".bak"
	err = os.Rename(myBinaryPath, myBinaryPathBak)
	if err != nil {
		return fmt.Errorf("Could not replace existing %v; do you need to be root?: %w",
			myBinaryPath, err)
	}
	err = os.Rename(tmpFile.Name(), myBinaryPath)
	if errors.Is(err, syscall.EXDEV) {
		// invalid cross device link occurs when rename() is attempted aross
		// different filesystems; copy instead
		err = ioutil.WriteFile(myBinaryPath, binaryContent, 0755)
		_ = os.Remove(tmpFile.Name())
	}
	if err != nil {
		err := fmt.Errorf("Could not replace existing %v; do you need to be root?: %w",
			myBinaryPath, err)
		_ = os.Rename(myBinaryPathBak, myBinaryPath)
		return err
	}
	_ = os.Remove(myBinaryPathBak)

	fmt.Printf("Upgrade %v to %v complete\n", myBinaryPath, latestVer)

	return nil
}

func checkAndPrintUpgradeWarning() bool {
	if versionText == DevVersionText {
		return false
	}
	latestVer, err := getLatestVersion()
	if err != nil {
		return false
	}
	if latestVer == versionText {
		return false
	}

	fmt.Fprintf(os.Stderr, "*WARN*: A new version of %v is available (%v). Please upgrade via '%v upgrade'.\n\n",
		CommandName, latestVer, CommandName)

	return true
}

// "--" is used to delineate between the user's prompt, and the reponse from
// OpenAI's GPT API after the user pressed the [TAB] key. this allows the user
// to simply press enter after viewing the response in order to execute the
// returned command(s). Additionally, this also allows both
// the prompt along with the response to be stored in .bash_history for
// repeatability.
// see bashgpt_autocomplete.sh before making changes here
func argsToPromptAndCmd(slice []string) (string, []string) {
	var promptStr string
	var cmdAndArgs []string
	foundDelim := false

	for _, s := range slice {
		if s == PromptDelim {
			foundDelim = true
			continue
		}

		if !foundDelim {
			promptStr += s + " "
		} else {
			cmdAndArgs = append(cmdAndArgs, s)
		}
	}

	return promptStr, cmdAndArgs
}

func execMain(cmdAndArgs []string) error {
	cmd := exec.Command(cmdAndArgs[0], cmdAndArgs[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func shMain(args []string) error {
	keyText, err := loadKey()
	if err != nil {
		return err
	}
	client := openai.NewClient(keyText)
	prompt, cmdAndArgs := argsToPromptAndCmd(args)

	if len(cmdAndArgs) != 0 {
		// execute the already returned response from the prompt
		return execMain(cmdAndArgs)
	}

	dialogue := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: SystemMsg},
		{Role: openai.ChatMessageRoleUser, Content: prompt},
	}

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: dialogue,
		},
	)
	if err != nil {
		return err
	}

	if len(resp.Choices) != 1 {
		return fmt.Errorf("Expected 1 response, got %v", len(resp.Choices))
	}
	cmdStr, err := parseResponse(resp.Choices[0].Message.Content)
	if err != nil {
		return err
	}

	fmt.Printf("%v", cmdStr)

	//	fmt.Fprintf(os.Stderr, "FULL RESPONSE: %v\n", resp.Choices[0].Message.Content)

	return nil
}

func parseResponse(resp string) (string, error) {
	if strings.Contains(resp, "```") {
		var sb strings.Builder
		isCmdText := false

		for _, respLine := range strings.Split(resp, "\n") {
			if strings.HasPrefix(respLine, "```") {
				isCmdText = !isCmdText
			} else if isCmdText {
				sb.WriteString(respLine + "\n")
			}
		}

		return sb.String(), nil
	}
	return resp, nil
}

//go:embed bashgpt_autocomplete.sh
var autocompleteScriptText string

func checkLatestAutocompleteScript() error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}
	existingScriptPath := filepath.Join(configDir, AutocompleteScript)
	existingScriptText, err := ioutil.ReadFile(existingScriptPath)
	if err == nil {
		if string(existingScriptText) == autocompleteScriptText {
			// already have the latest
			return nil
		}
		_ = os.Remove(existingScriptPath)
	}

	err = ioutil.WriteFile(existingScriptPath, []byte(autocompleteScriptText),
		0755)
	if err != nil {
		return fmt.Errorf("Could not update script %v: %w", existingScriptPath,
			err)
	}

	return nil
}

func main() {
	var args []string
	if len(os.Args) > 1 {
		args = os.Args[1:]
	}

	subCommandName := ""
	if len(args) > 0 {
		subCommandName = args[0]
	}
	exitStatus := 0

	if len(args) > 1 {
		args = args[1:]
	}

	if subCommandName != "upgrade" {
		checkAndPrintUpgradeWarning()
		_ = checkLatestAutocompleteScript()
	}

	var err error
	subCommand, ok := subCommandTab[subCommandName]
	if !ok {
		subCommand = helpMain
		exitStatus = 1
	}
	err = subCommand(args)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		exitStatus = 1
	}

	os.Exit(exitStatus)
}
