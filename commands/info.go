package commands

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/10gen/stitch-cli/app"
	"github.com/10gen/stitch-cli/local"
	"github.com/10gen/stitch-cli/ui"
	flag "github.com/ogier/pflag"
)

var info = &Command{
	Run:  infoRun,
	Name: "info",
	ShortUsage: `
USAGE:
    stitch info [--help] [--app <APP-ID>] [--json] [<TOP-LEVEL-SPEC> [<INNER-SPEC>...]]
`,
	LongUsage: `Show info about a particular app.

ARGS:
    <TOP-LEVEL-SPEC>
            The top level specifier, one of "group", "name", "id", "client-id",
            "clusters", "services", "pipelines", "values", or "authentication".
            Gives specific information on the given specifier.
   <INNER-SPEC>
            An inner specifier, according to whatever keys are made availables
            for the particular top level specifier:
            clusters: list of clusters
                - name of a cluster: MongoDB URI
            services: list of services
                - name of a service. Within a specified services:
                    - type: type of service
                    - name: name of the service
                    - webhooks: list of webhooks
                        - name of a webhook. Within a specified webhook:
                            - id: identifer of the webhook
                            - name: name of the webhook
                            - output: output type of the webhook
                            - pipeline: JSON of the webhook's pipeline
                    - rules: list of rules
                        - name of a rule. Within a specified webhook:
                            - name: name of the rule
                            - id: identifer of the rule
                            - rule: JSON of the complete rule
            pipelines: list of named pipelines
                - name of a pipeline. Within a specified pipeline:
                    - name: name of the pipeline
                    - id: identifer of the pipeline
                    - output: output type of the pipeline
                    - private: whether the pipeline is private
                    - skip-rules: whether the pipeline skips rules
                    - parameters: list of parameters to the pipeline
                    - can-evaluate: JSON of the evaluation condition on the pipeline
                    - pipeline: JSON of the pipeline
            - values: list of value names
                - name of value: assigned value
            - authentication: list of configure authentication providers
                - name of auth provider. Within a specified auth provider:
				    - type: type of auth provider
					- name: name of the auth provider
					- id: identifier of the auth provider
					- enabled: whether the auth provider is enabled
					- metadata: list of user metadata fields given by the auth provider
					- domain-restrictions: set of domains that users can authenticate with
					- redirect-URIs: the URIs that users are redirected to after successful authentication
					- config: JSON of other configuration options

OPTIONS:
    --app <APP-ID>
            Shows info for the specified app according to the stitch server.
            Leave unset to use local config.

    --json
            Show info in JSON form.
`,
}

var (
	infoFlagSet *flag.FlagSet

	flagInfoApp  string
	flagInfoJSON bool
)

func init() {
	infoFlagSet = info.InitFlags()
	infoFlagSet.StringVar(&flagInfoApp, "app", "", "")
	infoFlagSet.BoolVar(&flagInfoJSON, "json", false, "")
}

func infoRun() error {
	args := infoFlagSet.Args()
	if len(args) == 0 {
		return infoAll()
	}
	switch args[0] {
	case "clusters":
		return infoClusters(args[1:])
	case "services":
		return infoServices(args[1:])
	case "pipelines":
		return infoPipelines(args[1:])
	case "values":
		return infoValues(args[1:])
	case "authentication":
		return infoAuthentication(args[1:])
	default:
		if len(args) > 1 {
			return errUnknownArg(args[2])
		}
		return infoItem(args[0])
	}
}

func infoGetApp() (a app.App, isLocal bool, err error) {
	if flagInfoApp == "" {
		a, isLocal = local.GetApp()
		if !isLocal {
			err = errorf("no local project found and --app was not specified.")
		}
		return
	}
	// TODO: export based on flagInfoApp to create app.App
	err = errorf("could not find app %q", flagInfoApp)
	return
}

func infoAll() error {
	app, isLocal, err := infoGetApp()
	if err != nil {
		return err
	}

	var clusters, pipelines, values, authentication []string
	var services [][2]string

	for _, cluster := range app.Clusters {
		clusters = append(clusters, cluster.Name)
	}
	for _, pipeline := range app.Pipelines {
		pipelines = append(pipelines, pipeline.Name)
	}
	for _, value := range app.Values {
		values = append(values, value.Name)
	}
	for _, authProvider := range app.AuthProviders {
		authentication = append(authentication, authProvider.Name)
	}
	for _, service := range app.Services {
		services = append(services, [2]string{service.Type, service.Name})
	}

	if flagInfoJSON {
		servicesJSON := make([]interface{}, len(services))
		for i, service := range services {
			servicesJSON[i] = map[string]interface{}{
				"type": service[0],
				"name": service[1],
			}
		}
		obj := map[string]interface{}{
			"group":          app.Group,
			"name":           app.Name,
			"id":             app.ID,
			"clientId":       app.ClientID,
			"clusters":       clusters,
			"services":       servicesJSON,
			"pipelines":      pipelines,
			"values":         values,
			"authentication": authentication,
		}
		if isLocal {
			obj["local"] = "yes"
		}
		raw, _ := json.Marshal(obj)
		fmt.Printf("%s\n", raw)
	} else {
		var items []kv
		if isLocal {
			items = append(items, kv{key: "local", value: "yes"})
		}
		for i := range services {
			services[i][0] = ui.Color(ui.ServiceType, services[i][0])
		}
		items = append(items,
			kv{key: "group", value: ui.Color(ui.Group, app.Group)},
			kv{key: "name", value: app.Name},
			kv{key: "id", value: app.ID},
			kv{key: "client-id", value: ui.Color(ui.AppClientID, app.ClientID)},
			kv{key: "clusters", values: clusters},
			kv{key: "services", valuePairs: services},
			kv{key: "pipelines", values: pipelines},
			kv{key: "values", values: values},
			kv{key: "authentication", values: authentication},
		)
		printKV(items)
	}
	return nil
}

func infoItem(item string) error {
	app, _, err := infoGetApp()
	if err != nil {
		return err
	}
	var output string
	var v ui.Variant = ui.None
	switch item {
	case "group":
		output = app.Group
		v = ui.Group
	case "name":
		output = app.Name
	case "id":
		output = app.ID
	case "client-id":
		output = app.ClientID
		v = ui.AppClientID
	default:
		return errUnknownArg(item)
	}
	if flagInfoJSON {
		raw, _ := json.Marshal(output)
		fmt.Printf("%s\n", raw)
	} else {
		fmt.Println(ui.Color(v, output))
	}
	return nil
}

func infoClusters(args []string) error {
	var clusterName string
	if len(args) > 0 {
		clusterName = args[0]
		args = args[1:]
	}
	if len(args) > 0 {
		return errUnknownArg(args[0])
	}

	userApp, _, err := infoGetApp()
	if err != nil {
		return err
	}

	if clusterName == "" {
		clusters := make([]string, len(userApp.Clusters))
		for i, cluster := range userApp.Clusters {
			clusters[i] = cluster.Name
		}
		if flagInfoJSON {
			raw, _ := json.Marshal(clusters)
			fmt.Printf("%s\n", raw)
		} else {
			for _, cluster := range clusters {
				fmt.Println(cluster)
			}
		}
		return nil
	}

	var cluster app.Cluster
	for _, cluster = range userApp.Clusters {
		if cluster.Name == clusterName {
			break
		}
	}
	if cluster.Name != clusterName {
		return errorf("cluster %q not found.", clusterName)
	}
	if flagInfoJSON {
		raw, _ := json.Marshal(cluster.URI)
		fmt.Printf("%s\n", raw)
	} else {
		fmt.Println(cluster.URI)
	}
	return nil
}

func infoServices(args []string) error {
	if len(args) > 0 {
		return infoServicesParticular(args[0], args[1:])
	}
	app, _, err := infoGetApp()
	if err != nil {
		return err
	}

	if flagInfoJSON {
		services := make([]interface{}, len(app.Services))
		for i, service := range app.Services {
			services[i] = map[string]interface{}{
				"type": service.Type,
				"name": service.Name,
			}
		}
		raw, _ := json.Marshal(services)
		fmt.Printf("%s\n", raw)
	} else {
		services := make([][2]string, len(app.Services))
		for i, service := range app.Services {
			services[i] = [2]string{
				ui.Color(ui.ServiceType, service.Type),
				service.Name,
			}
		}
		printSingleKV(kv{valuePairs: services})
	}
	return nil
}

func infoServicesParticular(name string, args []string) error {
	userApp, _, err := infoGetApp()
	if err != nil {
		return err
	}

	var service app.Service
	for _, service = range userApp.Services {
		if service.Name == name {
			break
		}
	}
	if service.Name != name {
		return errorf("service %q not found", name)
	}

	if len(args) == 0 {
		webhooks := make([]string, len(service.Webhooks))
		for i, webhook := range service.Webhooks {
			webhooks[i] = webhook.Name
		}
		rules := make([]string, len(service.Rules))
		for i, rule := range service.Rules {
			rules[i] = rule.Name
		}
		if flagInfoJSON {
			obj := map[string]interface{}{
				"type":     service.Type,
				"name":     service.Name,
				"webhooks": webhooks,
				"rules":    rules,
			}
			raw, _ := json.Marshal(obj)
			fmt.Printf("%s\n", raw)
		} else {
			items := []kv{
				{key: "type", value: ui.Color(ui.ServiceType, service.Type)},
				{key: "name", value: service.Name},
				{key: "webhooks", values: webhooks},
				{key: "rules", values: rules},
			}
			printKV(items)
		}
		return nil
	}

	subcmd := args[0]
	args = args[1:]
	switch subcmd {
	case "type":
		if len(args) > 0 {
			return errUnknownArg(args[0])
		}
		output := service.Type
		if flagInfoJSON {
			raw, _ := json.Marshal(output)
			fmt.Printf("%s\n", raw)
		} else {
			output = ui.Color(ui.ServiceType, output)
			fmt.Println(output)
		}
		return nil
	case "name":
		if len(args) > 0 {
			return errUnknownArg(args[0])
		}
		output := service.Name
		if flagInfoJSON {
			raw, _ := json.Marshal(output)
			fmt.Printf("%s\n", raw)
		} else {
			fmt.Println(output)
		}
		return nil
	case "webhooks":
		return infoServicesParticularWebhooks(service, args)
	case "rules":
		return infoServicesParticularRules(service, args)
	default:
		return errUnknownArg(subcmd)
	}
}

func infoServicesParticularWebhooks(service app.Service, args []string) error {
	if len(args) == 0 {
		if flagInfoJSON {
			webhookNames := make([]string, len(service.Webhooks))
			for i, webhook := range service.Webhooks {
				webhookNames[i] = webhook.Name
			}
			raw, _ := json.Marshal(webhookNames)
			fmt.Printf("%s\n", raw)
		} else {
			for _, webhook := range service.Webhooks {
				fmt.Println(webhook.Name)
			}
		}
		return nil
	}

	name := args[0]
	args = args[1:]
	var webhook app.Webhook
	for _, webhook = range service.Webhooks {
		if webhook.Name == name {
			break
		}
	}
	if webhook.Name != name {
		return errorf("service webhook %q not found", name)
	}

	if len(args) == 0 {
		if flagInfoJSON {
			obj := map[string]interface{}{
				"name":     webhook.Name,
				"id":       webhook.ID,
				"output":   webhook.Output,
				"pipeline": json.RawMessage(webhook.Pipeline),
			}
			raw, _ := json.Marshal(obj)
			fmt.Printf("%s\n", raw)
		} else {
			items := []kv{
				{key: "name", value: webhook.Name},
				{key: "id", value: webhook.ID},
				{key: "output", value: webhook.Output},
				{key: "pipeline", value: "# add 'pipeline' subcommand to get this JSON array"},
			}
			printKV(items)
		}
		return nil
	}

	subcmd := args[0]
	args = args[1:]
	if len(args) > 0 {
		return errUnknownArg(args[0])
	}
	var output string
	switch subcmd {
	case "name":
		output = webhook.Name
	case "id":
		output = webhook.ID
	case "output":
		output = webhook.Output
	case "pipeline":
		output = webhook.Pipeline
		fmt.Println(output) // output is always JSON
		return nil
	default:
		return errUnknownArg(subcmd)
	}
	// output is an uncolored string
	if flagInfoJSON {
		raw, _ := json.Marshal(output)
		fmt.Printf("%s\n", raw)
	} else {
		fmt.Println(output)
	}
	return nil
}

func infoServicesParticularRules(service app.Service, args []string) error {
	if len(args) == 0 {
		if flagInfoJSON {
			ruleNames := make([]string, len(service.Rules))
			for i, rule := range service.Rules {
				ruleNames[i] = rule.Name
			}
			raw, _ := json.Marshal(ruleNames)
			fmt.Printf("%s\n", raw)
		} else {
			for _, rule := range service.Rules {
				fmt.Println(rule.Name)
			}
		}
		return nil
	}

	name := args[0]
	args = args[1:]
	var rule app.ServiceRule
	for _, rule = range service.Rules {
		if rule.Name == name {
			break
		}
	}
	if rule.Name != name {
		return errorf("service rule %q not found", name)
	}

	if len(args) == 0 {
		if flagInfoJSON {
			obj := map[string]interface{}{
				"name": rule.Name,
				"id":   rule.ID,
				"rule": json.RawMessage(rule.Rule),
			}
			raw, _ := json.Marshal(obj)
			fmt.Printf("%s\n", raw)
		} else {
			items := []kv{
				{key: "name", value: rule.Name},
				{key: "id", value: rule.ID},
				{key: "rule", value: "# add 'rule' subcommand to get this JSON document"},
			}
			printKV(items)
		}
		return nil
	}

	subcmd := args[0]
	args = args[1:]
	if len(args) > 0 {
		return errUnknownArg(args[0])
	}
	var output string
	switch subcmd {
	case "name":
		output = rule.Name
	case "id":
		output = rule.ID
	case "rule":
		output = rule.Rule
		fmt.Println(output) // output is always JSON
		return nil
	default:
		return errUnknownArg(subcmd)
	}
	// output is an uncolored string
	if flagInfoJSON {
		raw, _ := json.Marshal(output)
		fmt.Printf("%s\n", raw)
	} else {
		fmt.Println(output)
	}
	return nil
}

func infoPipelines(args []string) error {
	if len(args) > 0 {
		return infoPipelinesParticular(args[0], args[1:])
	}
	app, _, err := infoGetApp()
	if err != nil {
		return err
	}

	if flagInfoJSON {
		pipelines := make([]string, len(app.Pipelines))
		for i, pipeline := range app.Pipelines {
			pipelines[i] = pipeline.Name
		}
		raw, _ := json.Marshal(pipelines)
		fmt.Printf("%s\n", raw)
	} else {
		for _, pipeline := range app.Pipelines {
			fmt.Println(pipeline.Name)
		}
	}
	return nil
}

func infoPipelinesParticular(name string, args []string) error {
	userApp, _, err := infoGetApp()
	if err != nil {
		return err
	}

	var pipeline app.Pipeline
	for _, pipeline = range userApp.Pipelines {
		if pipeline.Name == name {
			break
		}
	}
	if pipeline.Name != name {
		return errorf("pipeline %q not found", name)
	}

	if len(args) == 0 {
		if flagInfoJSON {
			parameters := make([]interface{}, len(pipeline.Parameters))
			for i, parameter := range pipeline.Parameters {
				parameters[i] = map[string]interface{}{
					"name":     parameter.Name,
					"required": parameter.Required,
				}
			}
			obj := map[string]interface{}{
				"name":        pipeline.Name,
				"id":          pipeline.ID,
				"output":      pipeline.Output,
				"private":     pipeline.Private,
				"skipRules":   pipeline.SkipRules,
				"parameters":  parameters,
				"canEvaluate": json.RawMessage(pipeline.CanEvaluate),
				"pipeline":    json.RawMessage(pipeline.Pipeline),
			}
			raw, _ := json.Marshal(obj)
			fmt.Printf("%s\n", raw)
		} else {
			parameters := make([][2]string, len(pipeline.Parameters))
			for i, parameter := range pipeline.Parameters {
				optionality := "optional"
				if parameter.Required {
					optionality = "required"
				}
				parameters[i] = [2]string{
					ui.Color(ui.ParameterName, parameter.Name),
					optionality,
				}
			}
			items := []kv{
				{key: "name", value: pipeline.Name},
				{key: "id", value: pipeline.ID},
				{key: "output", value: pipeline.Output},
				{key: "private", value: ui.Color(ui.Boolean, strconv.FormatBool(pipeline.Private))},
				{key: "skip-rules", value: ui.Color(ui.Boolean, strconv.FormatBool(pipeline.SkipRules))},
				{key: "parameters", valuePairs: parameters},
				{key: "can-evaluate", value: "# add 'can-evaluate' subcommand to get this JSON document"},
				{key: "pipeline", value: "# add 'pipeline' subcommand to get this JSON document"},
			}
			printKV(items)
		}
		return nil
	}

	subcmd := args[0]
	args = args[1:]
	if len(args) > 0 {
		return errUnknownArg(args[0])
	}
	var output string
	switch subcmd {
	case "name":
		output = pipeline.Name
	case "id":
		output = pipeline.ID
	case "output":
		output = pipeline.Output
	case "private":
		if flagInfoJSON {
			raw, _ := json.Marshal(pipeline.Private)
			fmt.Printf("%s\n", raw)
		} else {
			output = ui.Color(ui.Boolean, strconv.FormatBool(pipeline.Private))
			fmt.Println(output)
		}
		return nil
	case "skip-rules":
		if flagInfoJSON {
			raw, _ := json.Marshal(pipeline.SkipRules)
			fmt.Printf("%s\n", raw)
		} else {
			output = ui.Color(ui.Boolean, strconv.FormatBool(pipeline.SkipRules))
			fmt.Println(output)
		}
		return nil
	case "parameters":
		if flagInfoJSON {
			parameters := make([]interface{}, len(pipeline.Parameters))
			for i, parameter := range pipeline.Parameters {
				parameters[i] = map[string]interface{}{
					"name":     parameter.Name,
					"required": parameter.Required,
				}
			}
			raw, _ := json.Marshal(parameters)
			fmt.Printf("%s\n", raw)
		} else {
			parameters := make([][2]string, len(pipeline.Parameters))
			for i, parameter := range pipeline.Parameters {
				optionality := "optional"
				if parameter.Required {
					optionality = "required"
				}
				parameters[i] = [2]string{
					ui.Color(ui.ParameterName, parameter.Name),
					optionality,
				}
			}
			printSingleKV(kv{valuePairs: parameters})
		}
		return nil
	case "can-evaluate":
		output = pipeline.CanEvaluate
		fmt.Println(output) // output is always JSON
		return nil
	case "pipeline":
		output = pipeline.Pipeline
		fmt.Println(output) // output is always JSON
		return nil
	default:
		return errUnknownArg(subcmd)
	}
	// output is an uncolored string
	if flagInfoJSON {
		raw, _ := json.Marshal(output)
		fmt.Printf("%s\n", raw)
	} else {
		fmt.Println(output)
	}
	return nil
}

func infoValues(args []string) error {
	var valueName string
	if len(args) > 0 {
		valueName = args[0]
		args = args[1:]
	}
	if len(args) > 0 {
		return errUnknownArg(args[0])
	}

	userApp, _, err := infoGetApp()
	if err != nil {
		return err
	}

	if valueName == "" {
		var valueNames []string
		for _, value := range userApp.Values {
			valueNames = append(valueNames, value.Name)
		}
		if flagInfoJSON {
			raw, _ := json.Marshal(valueNames)
			fmt.Printf("%s\n", raw)
		} else {
			for _, valueName := range valueNames {
				fmt.Println(valueName)
			}
		}
		return nil
	}

	var value app.Value
	for _, value = range userApp.Values {
		if value.Name == valueName {
			break
		}
	}
	if value.Name != valueName {
		return errorf("value %q not found.", valueName)
	}
	if !flagInfoJSON {
		if s, ok := value.Value.(string); ok {
			fmt.Println(s)
			return nil
		}
	}
	raw, _ := json.Marshal(value.Value)
	fmt.Printf("%s\n", raw)
	return nil
}

func infoAuthentication(args []string) error {
	// TODO: make sure this displays the appropriate fields.
	// Perhaps the displayed fields should depend on the authProvider.
	var authProviderName string
	if len(args) > 0 {
		authProviderName = args[0]
		args = args[1:]
	}

	userApp, _, err := infoGetApp()
	if err != nil {
		return err
	}

	if authProviderName == "" {
		if len(args) > 0 {
			return errUnknownArg(args[0])
		}
		if flagInfoJSON {
			auths := make([]interface{}, len(userApp.AuthProviders))
			for i, auth := range userApp.AuthProviders {
				auths[i] = map[string]interface{}{
					"type": auth.Type,
					"name": auth.Name,
				}
			}
			raw, _ := json.Marshal(auths)
			fmt.Printf("%s\n", raw)
		} else {
			auths := make([][2]string, len(userApp.AuthProviders))
			for i, auth := range userApp.AuthProviders {
				auths[i] = [2]string{
					ui.Color(ui.AuthProviderType, auth.Type),
					auth.Name,
				}
			}
			printSingleKV(kv{valuePairs: auths})
		}
		return nil
	}

	var authProvider app.AuthProvider
	for _, authProvider = range userApp.AuthProviders {
		if authProvider.Name == authProviderName {
			break
		}
	}
	if authProvider.Name != authProviderName {
		return errorf("authentication %q not found.", authProviderName)
	}

	if len(args) == 0 {
		if flagInfoJSON {
			obj := map[string]interface{}{
				"type":                authProvider.Type,
				"name":                authProvider.Name,
				"id":                  authProvider.ID,
				"enabled":             authProvider.Enabled,
				"metadata":            authProvider.Metadata,
				"domain-restrictions": authProvider.DomainRestrictions,
				"redirect-URIs":       authProvider.RedirectURIs,
				"config":              json.RawMessage(authProvider.Config),
			}
			raw, _ := json.Marshal(obj)
			fmt.Printf("%s\n", raw)
		} else {
			items := []kv{
				{key: "type", value: ui.Color(ui.AuthProviderType, authProvider.Type)},
				{key: "name", value: authProvider.Name},
				{key: "id", value: authProvider.ID},
				{key: "enabled", value: ui.Color(ui.Boolean, strconv.FormatBool(authProvider.Enabled))},
				{key: "config", value: "# add 'config' subcommand to get this JSON document"},
				{key: "metadata", values: authProvider.Metadata},
				{key: "domain-restrictions", values: authProvider.DomainRestrictions},
				{key: "redirect-URIs", values: authProvider.RedirectURIs},
			}
			printKV(items)
		}
		return nil
	}

	subcmd := args[0]
	args = args[1:]
	if len(args) > 0 {
		return errUnknownArg(args[0])
	}
	var output string
	switch subcmd {
	case "type":
		output = authProvider.Type
		if flagInfoJSON {
			raw, _ := json.Marshal(output)
			fmt.Printf("%s\n", raw)
		} else {
			output = ui.Color(ui.AuthProviderType, output)
			fmt.Println(output)
		}
		return nil
	case "name":
		output = authProvider.Name
		if flagInfoJSON {
			raw, _ := json.Marshal(output)
			fmt.Printf("%s\n", raw)
		} else {
			fmt.Println(output)
		}
		return nil
	case "id":
		output = authProvider.ID
		if flagInfoJSON {
			raw, _ := json.Marshal(output)
			fmt.Printf("%s\n", raw)
		} else {
			fmt.Println(output)
		}
		return nil
	case "enabled":
		if flagInfoJSON {
			raw, _ := json.Marshal(authProvider.Enabled)
			fmt.Printf("%s\n", raw)
		} else {
			output = ui.Color(ui.Boolean, strconv.FormatBool(authProvider.Enabled))
			fmt.Println(output)
		}
		return nil
	case "metadata":
		if flagInfoJSON {
			raw, _ := json.Marshal(authProvider.Metadata)
			fmt.Printf("%s\n", raw)
		} else {
			printSingleKV(kv{values: authProvider.Metadata})
		}
		return nil
	case "domain-restrictions":
		if flagInfoJSON {
			raw, _ := json.Marshal(authProvider.DomainRestrictions)
			fmt.Printf("%s\n", raw)
		} else {
			printSingleKV(kv{values: authProvider.DomainRestrictions})
		}
		return nil
	case "redirect-URIs":
		if flagInfoJSON {
			raw, _ := json.Marshal(authProvider.RedirectURIs)
			fmt.Printf("%s\n", raw)
		} else {
			printSingleKV(kv{values: authProvider.RedirectURIs})
		}
		return nil
	case "config":
		output = authProvider.Config
		fmt.Println(output) // output is always JSON
		return nil
	default:
		return errUnknownArg(subcmd)
	}
}