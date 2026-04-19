# Capturing the Instacart history GraphQL hashes

The `history sync` command needs two Instacart GraphQL operation hashes:

- `BuyItAgainPage` — the aggregated "frequently bought" + purchase history feed
- `CustomerOrderHistory` — the per-order list with items

They ship **empty** in this CLI today because the hashes are specific to your
Instacart web bundle and rotate whenever Instacart ships a new frontend. Once
you fill them in, `history sync` works and stays working until the next bundle
rotation.

## Two-minute walkthrough

1. **Log in to Instacart** in Chrome, then open the **Orders** page at
   <https://www.instacart.com/store/account/orders>. Keep the page open.
2. **Open DevTools** (`Cmd+Option+I` on macOS / `F12` on Windows/Linux) and
   switch to the **Network** tab. Filter by `graphql` in the filter box.
3. **Reload the page.** A burst of GraphQL requests fires. Click one at a
   time and look at the **Request Payload** tab — you want two requests:

   - One with `operationName: "BuyItAgainPage"` (or similar — Instacart
     occasionally renames; `BuyItAgain`, `BuyItAgainPageQuery`, and
     `BuyItAgainProducts` are all candidates)
   - One with `operationName: "CustomerOrderHistory"` (or `OrdersHistory`,
     `UserOrders`, `OrderCollectionQuery`)

   The exact names sometimes drift; pick whichever clearly returns your
   order list.

4. **Copy the `sha256Hash` value** from each request. It lives under
   `extensions.persistedQuery.sha256Hash` in the request payload. Each is
   a 64-character hex string.

5. **Drop the hashes into the Go source** at
   `internal/instacart/ops.go`. Replace the empty `Hash: ""` fields on the
   `BuyItAgainPage` and `CustomerOrderHistory` entries.

6. **Rebuild** the CLI:

   ```bash
   go install github.com/mvanhorn/printing-press-library/library/commerce/instacart/cmd/instacart-pp-cli@latest
   ```

7. **Re-seed the local hash cache**:

   ```bash
   instacart capture
   instacart history sync
   ```

## Alternative: remote registry

If you don't want to edit Go source, drop the hashes into the community
registry at `library/commerce/instacart/hashes.json` in this repo and open
a PR. Every user running `instacart capture --remote` will pick up the
new hashes at their next refresh. Template:

```json
{
  "version": 1,
  "updated_at": "2026-04-18T00:00:00Z",
  "operations": {
    "BuyItAgainPage": "<paste your 64-char hex>",
    "CustomerOrderHistory": "<paste your 64-char hex>"
  }
}
```

## Operation name mismatches

If DevTools shows an `operationName` that differs from the two defaults above
(e.g. Instacart renamed the underlying query), rename the keys in
`internal/instacart/ops.go` to match. The rest of the history code
references them by those exact names, so keep them in sync.

## Verifying

After capturing and rebuilding:

```bash
instacart doctor     # shows history: enabled=true|false based on hash presence
instacart history sync
instacart history list --limit 10
```

If `doctor` still reports `history: hashes not yet captured`, the Go file
edit or the remote-registry merge didn't take effect — re-check the two
entries in `internal/instacart/ops.go` or re-run `instacart capture --remote`.
