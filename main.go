package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := newManifestAnnotatorCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
	}
}

func newManifestAnnotatorCommand() *cobra.Command {
	var opts manifestAnnotator
	cmd := &cobra.Command{
		Use:   "manifest-annotator FILENAME ANNOTATION VALUE [OPTS]",
		Short: "Add/Update annotations in a yaml manifest file",
		Long: `Updates a yaml manifest file without changing the file's structure,
removing comments, etc. Supports files with multiple manifests.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 3 {
				cmd.Usage()
				return nil
			}
			opts.FileName = args[0]
			opts.Annotation = args[1]
			opts.Value = args[2]
			return opts.Run()
		},
	}
	cmd.Flags().StringVar(&opts.Name, "name", "", "[optional] Only annotate manifests with this name")
	cmd.Flags().StringVar(&opts.Namespace, "namespace", "", "[optional] Only annotate manifests with this namespace")
	cmd.Flags().StringVar(&opts.Kind, "kind", "", "[optional] Only annotate manifests with this kind")
	cmd.Flags().StringVar(&opts.GroupVersion, "groupVersion", "", "[optional] Only annotate manifests with this group and version")
	return cmd
}
