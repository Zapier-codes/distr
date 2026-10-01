package cmd

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"

	"github.com/distr-sh/distr/internal/buildnotify"
	"github.com/distr-sh/distr/internal/env"
	"github.com/spf13/cobra"
)

// NewNovuTestCommand sends one sample "your app is ready" mail through the Novu instance this deployment is
// configured with (leaf f.xi), so the operator can see that the key, the URL, the workflow and the sender identity
// work before a real requester depends on them. It needs the same environment as `serve`, but connects to nothing
// except Novu.
func NewNovuTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "novu-test <email>",
		Short: "send a sample \"your app is ready\" mail through the configured Novu instance",
		Long: "Triggers the build-ready workflow once with a sample payload for the given address.\n" +
			"Uses NOVU_API_KEY, NOVU_API_URL and NOVU_BUILD_READY_WORKFLOW_ID, and reads no tenant record.\n" +
			"Success means Novu accepted the trigger; check the inbox to confirm the mail arrived.",
		Args: cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			env.Initialize()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			address, err := mail.ParseAddress(args[0])
			if err != nil || address.Address != args[0] {
				return fmt.Errorf("%q is not a plain email address", args[0])
			}
			config := env.Novu()
			if config == nil {
				return errors.New("NOVU_API_KEY is not set, or NOVU_API_URL is not an http(s) URL")
			}

			out := cmd.OutOrStdout()
			// Never print the key: only where the trigger goes and which workflow it starts.
			novuHost := config.APIURL
			if u, err := url.Parse(config.APIURL); err == nil {
				novuHost = u.Host
			}
			fmt.Fprintf(out, "Novu: %v, workflow %v\n", novuHost, config.BuildReadyWorkflowID)

			if err := buildnotify.SendTestMail(
				cmd.Context(), address.Address, novuTestBaseURL(env.Host(), env.HostScheme()),
			); err != nil {
				return err
			}
			fmt.Fprintf(out, "Novu accepted the trigger for %v. Check that inbox, and the spam folder, for the mail.\n",
				address.Address)
			return nil
		},
	}
}

// novuTestBaseURL is the origin of this deployment, from DISTR_HOST. It repeats what the handlers package does for the
// links of real mails, because that helper is not exported and this command is the only other place that needs it.
func novuTestBaseURL(host string, scheme env.URLScheme) string {
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	if strings.Contains(host, "://") {
		return host
	}
	return fmt.Sprintf("%v://%v", scheme, host)
}

func init() {
	RootCommand.AddCommand(NewNovuTestCommand())
}
