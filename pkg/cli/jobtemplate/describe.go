package jobtemplate

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"volcano.sh/apis/pkg/apis/flow/v1alpha1"
	"volcano.sh/apis/pkg/client/clientset/versioned"
	"volcano.sh/volcano/pkg/cli/util"
)

type describeFlags struct {
	util.CommonFlags

	// Name is name of job template
	Name string
	// Namespace is namespace of job template
	Namespace string
	// Format print format: yaml or json format
	Format string
}

var describeJobTemplateFlags = &describeFlags{}

// InitDescribeFlags is used to init all flags.
func InitDescribeFlags(cmd *cobra.Command) {
	util.InitFlags(cmd, &describeJobTemplateFlags.CommonFlags)
	cmd.Flags().StringVarP(&describeJobTemplateFlags.Name, "name", "N", "", "the name of job template")
	cmd.Flags().StringVarP(&describeJobTemplateFlags.Namespace, "namespace", "n", "default", "the namespace of job template")
	cmd.Flags().StringVarP(&describeJobTemplateFlags.Format, "format", "o", "yaml", "the format of output")
}

// DescribeJobTemplate is used to get the particular job template details.
func DescribeJobTemplate(ctx context.Context) error {
	config, err := util.BuildConfig(describeJobTemplateFlags.Master, describeJobTemplateFlags.Kubeconfig)
	if err != nil {
		return err
	}
	jobTemplateClient := versioned.NewForConfigOrDie(config)

	// Get job template list detail
	if describeJobTemplateFlags.Name == "" {
		jobTemplates, err := jobTemplateClient.FlowV1alpha1().JobTemplates(describeJobTemplateFlags.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, jobTemplate := range jobTemplates.Items {
			PrintJobTemplateDetail(&jobTemplate, describeJobTemplateFlags.Format)
		}
		// Get job template detail
	} else {
		jobTemplate, err := jobTemplateClient.FlowV1alpha1().JobTemplates(describeJobTemplateFlags.Namespace).Get(ctx, describeJobTemplateFlags.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		PrintJobTemplateDetail(jobTemplate, describeJobTemplateFlags.Format)
	}

	return nil
}

// PrintJobTemplateDetail print job template details
func PrintJobTemplateDetail(jobTemplate *v1alpha1.JobTemplate, format string) {
	switch format {
	case "json":
		printJSON(jobTemplate)
	case "yaml":
		printYAML(jobTemplate)
	default:
		log.Fatalf("Unsupported format: %s", format)
	}
}

func printJSON(jobTemplate *v1alpha1.JobTemplate) {
	b, err := json.MarshalIndent(jobTemplate, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
	fmt.Println("---------------------------------")
}

func printYAML(jobTemplate *v1alpha1.JobTemplate) {
	b, err := yaml.Marshal(jobTemplate)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
	fmt.Println("---------------------------------")
}
