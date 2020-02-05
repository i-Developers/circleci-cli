package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/CircleCI-Public/circleci-cli/api"
	"github.com/CircleCI-Public/circleci-cli/client"
	"github.com/CircleCI-Public/circleci-cli/git"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"

	"github.com/CircleCI-Public/circleci-cli/settings"
	"github.com/spf13/cobra"
)

type Organization struct {
	name     string
	provider git.VcsType
}

// TODO: initialize the client in preRun
// This will allow us to fill in values passed on the command line like --debug

func newContextCommand(config *settings.Config) *cobra.Command {

	var organization Organization
	if vcs, org, err := git.InferOrganizationFromGitRemotes(); err == nil {
		organization.name = org
		organization.provider = vcs
	}

	command := &cobra.Command{
		Use:   "context",
		Short: "Contexts provide a mechanism for securing and sharing environment variables across projects. The environment variables are defined as name/value pairs and are injected at runtime.",
	}

	client := client.NewClient(config.Host, config.Endpoint, config.Token, true) //config.Debug)

	listCommand := &cobra.Command{
		Short: "List contexts",
		Use:   "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listContexts(client, organization)
		},
	}

	showContextCommand := &cobra.Command{
		Short: "Show a context",
		Use:   "show <name>",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showContext(client, organization, args[0])
		},
		Args: cobra.ExactArgs(1),
	}

	storeCommand := &cobra.Command{
		Short: "Store an new secret in the named context. The value is read from stdin.",
		Use:   "store <context name> <variable name>",
		RunE: func(cmd *cobra.Command, args []string) error {
			return storeEnvVar(client, organization, args[0], args[1])
		},
		Args: cobra.ExactArgs(2),
	}

	removeCommand := &cobra.Command{
		Short: "Remove a secret from the named context",
		Use:   "remove <context name> <variable name>",
		RunE: func(cmd *cobra.Command, args []string) error {
			return removeEnvVar(client, organization, args[0], args[1])
		},
		Args: cobra.ExactArgs(2),
	}

	createContextCommand := &cobra.Command{
		Short: "Create a new context",
		Use:   "create <context name>",
		RunE: func(cmd *cobra.Command, args []string) error {
			return createContext(client, organization, args[0])
		},
		Args: cobra.ExactArgs(1),
	}

	deleteContextCommand := &cobra.Command{
		Short: "Delete the named context",
		Use:   "delete <context name>",
		RunE: func(cmd *cobra.Command, args []string) error {
			return deleteContext(client, organization, args[0])
		},
		Args: cobra.ExactArgs(1),
	}

	command.PersistentFlags().StringVar(&organization.name, "organization", organization.name, "The organization to operate on.")

	command.AddCommand(listCommand)
	command.AddCommand(showContextCommand)
	command.AddCommand(storeCommand)
	command.AddCommand(removeCommand)
	command.AddCommand(createContextCommand)
	command.AddCommand(deleteContextCommand)

	return command
}

func listContexts(client *client.Client, org Organization) error {

	contexts, err := api.ListContexts(client, org.name, org.provider)

	if err != nil {
		return err

	}

	table := tablewriter.NewWriter(os.Stdout)

	table.SetHeader([]string{"Provider", "Organization", "Name", "Created At"})

	for _, context := range contexts.Organization.Contexts.Edges {

		table.Append([]string{
			string(org.provider),
			org.name,
			context.Node.Name,
			context.Node.CreatedAt,
		})
	}
	table.Render()

	return nil
}

func contextByName(client *client.Client, org Organization, contextName string) (*api.Context, error) {

	contexts, err := api.ListContexts(client, org.name, org.provider)

	if err != nil {
		return nil, err
	}

	for _, c := range contexts.Organization.Contexts.Edges {
		if c.Node.Name == contextName {
			return &c.Node, nil
		}
	}

	return nil, fmt.Errorf("Could not find a context named '%s' in the '%s' organization.", contextName, org)
}

func showContext(client *client.Client, org Organization, contextName string) error {

	context, err := contextByName(client, org, contextName)

	if err != nil {
		return err
	}

	fmt.Printf("Context: %s\n", context.Name)

	table := tablewriter.NewWriter(os.Stdout)

	table.SetHeader([]string{"Environment Variable", "Value"})

	for _, envVar := range context.Resources {
		table.Append([]string{envVar.Variable, "••••" + envVar.TruncatedValue})
	}
	table.Render()

	return nil
}

func readSecretValue() (string, error) {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		bytes, err := ioutil.ReadAll(os.Stdin)
		return string(bytes), err
	} else {
		fmt.Print("Enter secret value and press enter: ")
		reader := bufio.NewReader(os.Stdin)
		str, err := reader.ReadString('\n')
		return strings.TrimRight(str, "\n"), err
	}
}

func createContext(client *client.Client, org Organization, contextName string) error {
	return api.CreateContext(client, contextName, org.name, org.provider)
}

func removeEnvVar(client *client.Client, org Organization, contextName, varName string) error {
	context, err := contextByName(client, org, contextName)
	if err != nil {
		return err
	}
	return api.DeleteEnvironmentVariable(client, context.ID, varName)
}

func storeEnvVar(client *client.Client, org Organization, contextName, varName string) error {

	context, err := contextByName(client, org, contextName)

	if err != nil {
		return err
	}
	secretValue, err := readSecretValue()

	if err != nil {
		return errors.Wrap(err, "Failed to read secret value from stdin")
	}

	return api.StoreEnvironmentVariable(client, context.ID, varName, secretValue)
}

func deleteContext(client *client.Client, org Organization, contextName string) error {

	context, err := contextByName(client, org, contextName)

	if err != nil {
		return err
	}

	return api.DeleteContext(client, context.ID)

}
