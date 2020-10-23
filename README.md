# manifest-annotator
Utility to add an annotation to openshift manifest files

```
Updates a yaml manifest file without changing the file's structure,
removing comments, etc. Supports files with multiple manifests.

Usage:
  manifest-annotator FILENAME ANNOTATION VALUE [OPTS] [flags]

Flags:
      --groupVersion string   [optional] Only annotate manifests with this group and version
  -h, --help                  help for manifest-annotator
      --kind string           [optional] Only annotate manifests with this kind
      --name string           [optional] Only annotate manifests with this name
      --namespace string      [optional] Only annotate manifests with this namespace
```
