package cmd

import (
	"encoding/json"
	"io/ioutil"

	"github.com/spf13/cobra"
	"github.com/treeverse/lakefs/pkg/api"
)

var metadataDumpTemplate = `
{{ .Response | json }}
`

var refsRestoreSuccess = `
{{ "All references restored successfully!" | green }}
`

var refsRestoreCmd = &cobra.Command{
	Use:   "refs-restore <repository uri>",
	Short: "Restores refs (branches, commits, tags) from the underlying object store to a bare repository",
	Long: `restores refs (branches, commits, tags) from the underlying object store to a bare repository.

This command is expected to run on a bare repository (i.e. one created with 'lakectl repo create-bare').
Since a bare repo is expected, in case of transient failure, delete the repository and recreate it as bare and retry.`,
	Example: "aws s3 cp s3://bucket/_lakefs/refs_manifest.json - | lakectl refs-load lakefs://my-bare-repository --manifest -",
	Hidden:  true,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoURI := MustParseRepoURI("repository", args[0])
		Fmt("Repository: %s\n", repoURI.String())
		manifestFileName, _ := cmd.Flags().GetString("manifest")
		fp := OpenByPath(manifestFileName)
		defer func() {
			_ = fp.Close()
		}()

		// read and parse the JSON
		data, err := ioutil.ReadAll(fp)
		if err != nil {
			DieErr(err)
		}
		var manifest api.RefsDump
		err = json.Unmarshal(data, &manifest)
		if err != nil {
			DieErr(err)
		}
		// execute the restore operation
		client := getClient()
		resp, err := client.RestoreRefsWithResponse(cmd.Context(), repoURI.Repository, api.RestoreRefsJSONRequestBody(manifest))
		DieOnResponseError(resp, err)
		Write(refsRestoreSuccess, nil)
	},
}

var refsDumpCmd = &cobra.Command{
	Use:    "refs-dump <repository uri>",
	Short:  "Dumps refs (branches, commits, tags) to the underlying object store",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoURI := MustParseRepoURI("repository", args[0])
		Fmt("Repository: %s\n", repoURI.String())
		client := getClient()
		resp, err := client.DumpRefsWithResponse(cmd.Context(), repoURI.Repository)
		DieOnResponseError(resp, err)

		Write(metadataDumpTemplate, struct {
			Response interface{}
		}{resp.JSON201})
	},
}

//nolint:gochecknoinits
func init() {
	rootCmd.AddCommand(refsDumpCmd)
	rootCmd.AddCommand(refsRestoreCmd)

	refsRestoreCmd.Flags().String("manifest", "", "path to a refs manifest json file (as generated by `refs-dump`). Alternatively, use \"-\" to read from stdin")
	_ = refsRestoreCmd.MarkFlagRequired("manifest")
}
