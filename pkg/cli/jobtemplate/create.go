package jobtemplate

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	flowv1alpha1 "volcano.sh/apis/pkg/apis/flow/v1alpha1"
	"volcano.sh/apis/pkg/client/clientset/versioned"
	"volcano.sh/volcano/pkg/cli/util"
)

type createFlags struct {
	util.CommonFlags
	// FilePath is the file path of job template.
	FilePath string
}

var createJobTemplateFlags = &createFlags{}

// InitCreateFlags is used to init all flags during queue creating.
func InitCreateFlags(cmd *cobra.Command) {
	util.InitFlags(cmd, &createJobTemplateFlags.CommonFlags)
	cmd.Flags().StringVarP(&createJobTemplateFlags.FilePath, "file", "f", "", "the path to the YAML file containing the job template")
}

// CreateJobTemplate create a job template.
func CreateJobTemplate(ctx context.Context) error {
	config, err := util.BuildConfig(createJobTemplateFlags.Master, createJobTemplateFlags.Kubeconfig)
	if err != nil {
		return err
	}

	// Read YAML data from a file.
	yamlData, err := os.ReadFile(createJobTemplateFlags.FilePath)
	if err != nil {
		return err
	}
	// Split YAML data into individual documents.
	yamlDocs := strings.Split(string(yamlData), "---")

	jobTemplateClient := versioned.NewForConfigOrDie(config)
	createdCount := 0
	for _, doc := range yamlDocs {
		// Skip empty documents or documents with only whitespace.
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		// Parse each YAML document into a JobTemplate object.
		obj := &flowv1alpha1.JobTemplate{}
		if err = yaml.Unmarshal([]byte(doc), obj); err != nil {
			return err
		}
		// Set the namespace if it's not specified.
		if obj.Namespace == "" {
			obj.Namespace = "default"
		}

		_, err = jobTemplateClient.FlowV1alpha1().JobTemplates(obj.Namespace).Create(ctx, obj, metav1.CreateOptions{})
		if err == nil {
			fmt.Printf("Created JobTemplate: %s/%s\n", obj.Namespace, obj.Name)
			createdCount++
		} else {
			fmt.Printf("Failed to create JobTemplate: %v\n", err)
		}
	}

	return nil
}
