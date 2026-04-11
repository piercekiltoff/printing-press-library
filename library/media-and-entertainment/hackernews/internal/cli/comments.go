package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newCommentsCmd(flags *rootFlags) *cobra.Command {
	var flagFlat bool
	var flagDepth int
	var flagAuthor string
	var flagSince string
	var flagLimit int

	cmd := &cobra.Command{
		Use:   "comments <id>",
		Short: "Read comment threads with indentation, filtering, and flat mode",
		Example: `  # Read comments on a story
  hackernews-pp-cli comments 12345678

  # Flat mode for piping
  hackernews-pp-cli comments 12345678 --flat

  # Only 2 levels deep
  hackernews-pp-cli comments 12345678 --depth 2

  # Filter to one author
  hackernews-pp-cli comments 12345678 --author dang

  # JSON output
  hackernews-pp-cli comments 12345678 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var itemID int
			if _, err := fmt.Sscanf(args[0], "%d", &itemID); err != nil {
				return usageErr(fmt.Errorf("invalid item ID %q", args[0]))
			}

			// Parse since if given
			var sinceCutoff time.Time
			if flagSince != "" {
				dur, err := parseDuration(flagSince)
				if err != nil {
					return usageErr(err)
				}
				sinceCutoff = time.Now().Add(-dur)
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Fetching comments...\n")

			// Fetch root item
			root, err := fetchFirebaseItem(flags, itemID)
			if err != nil {
				return classifyAPIError(err)
			}

			// Recursively fetch comments
			type commentNode struct {
				item  map[string]any
				depth int
			}

			var allComments []commentNode
			var fetchComments func(parentIDs []int, depth int)
			fetchComments = func(parentIDs []int, depth int) {
				if flagDepth > 0 && depth > flagDepth {
					return
				}
				if len(allComments) >= 500 { // safety limit
					return
				}

				items, err := fetchFirebaseItems(flags, parentIDs, len(parentIDs))
				if err != nil {
					return
				}

				for _, item := range items {
					if getString(item, "type") != "comment" {
						continue
					}
					if getString(item, "deleted") == "true" || getString(item, "dead") == "true" {
						continue
					}

					// Apply author filter
					if flagAuthor != "" && getString(item, "by") != flagAuthor {
						// Still recurse into children in case they match
						kids := getIntSlice(item, "kids")
						if len(kids) > 0 {
							fetchComments(kids, depth+1)
						}
						continue
					}

					// Apply since filter
					if !sinceCutoff.IsZero() {
						t := getInt(item, "time")
						if t > 0 && time.Unix(int64(t), 0).Before(sinceCutoff) {
							continue
						}
					}

					allComments = append(allComments, commentNode{item: item, depth: depth})

					kids := getIntSlice(item, "kids")
					if len(kids) > 0 {
						fetchComments(kids, depth+1)
					}
				}
			}

			kids := getIntSlice(root, "kids")
			fetchComments(kids, 1)

			// Apply limit
			if flagLimit > 0 && len(allComments) > flagLimit {
				allComments = allComments[:flagLimit]
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "%d comments\n", len(allComments))

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				result := make([]map[string]any, 0, len(allComments))
				for _, c := range allComments {
					entry := map[string]any{
						"id":     getInt(c.item, "id"),
						"by":     getString(c.item, "by"),
						"text":   stripHTML(getString(c.item, "text")),
						"time":   getInt(c.item, "time"),
						"depth":  c.depth,
						"parent": getInt(c.item, "parent"),
					}
					result = append(result, entry)
				}
				data, _ := json.Marshal(result)
				return printOutput(cmd.OutOrStdout(), json.RawMessage(data), true)
			}

			if flagFlat {
				for _, c := range allComments {
					text := stripHTML(getString(c.item, "text"))
					author := getString(c.item, "by")
					fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", author, text)
				}
				return nil
			}

			// Threaded display
			for _, c := range allComments {
				indent := strings.Repeat("  ", c.depth-1)
				author := getString(c.item, "by")
				text := stripHTML(getString(c.item, "text"))
				t := getInt(c.item, "time")
				age := ""
				if t > 0 {
					age = timeAgo(time.Unix(int64(t), 0))
				}

				fmt.Fprintf(cmd.OutOrStdout(), "%s%s %s\n", indent, bold(author), age)
				// Word-wrap text at ~80 chars minus indent
				maxWidth := 80 - len(indent) - 2
				if maxWidth < 40 {
					maxWidth = 40
				}
				wrapped := wordWrap(text, maxWidth)
				for _, line := range strings.Split(wrapped, "\n") {
					fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", indent, line)
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagFlat, "flat", false, "Flatten output (one comment per line, no indentation)")
	cmd.Flags().IntVar(&flagDepth, "depth", 0, "Max nesting level (0 = unlimited)")
	cmd.Flags().StringVar(&flagAuthor, "author", "", "Filter to comments by this author")
	cmd.Flags().StringVar(&flagSince, "since", "", "Only comments newer than (e.g., 1h, 24h, 7d)")
	cmd.Flags().IntVar(&flagLimit, "limit", 0, "Maximum comments to show (0 = unlimited)")

	return cmd
}

// wordWrap breaks text into lines of at most maxWidth characters.
func wordWrap(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return text
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}
	var lines []string
	currentLine := words[0]
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) > maxWidth {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine += " " + word
		}
	}
	lines = append(lines, currentLine)
	return strings.Join(lines, "\n")
}
