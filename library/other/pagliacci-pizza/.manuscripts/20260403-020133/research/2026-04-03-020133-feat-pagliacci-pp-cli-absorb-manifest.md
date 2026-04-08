# Pagliacci Pizza CLI Absorb Manifest

## Ecosystem Scan Results

| # | Tool | Type | Features | Stars/Adoption |
|---|------|------|----------|---------------|

No existing CLI tools, MCP servers, Claude Code plugins, or community wrappers found for the Pagliacci Pizza API.

## Absorbed (match or beat everything that exists)

No existing tools to absorb from. This is a greenfield CLI.

| # | Feature | Source | Our Implementation | Added Value |
|---|---------|--------|-------------------|-------------|
| 1 | List stores | API discovery (sniff) | `stores list` + `--json --select` | First CLI access to store data |
| 2 | Browse menu | API discovery (sniff) | `menu cache <storeId>` + `--json` | Full menu in terminal |
| 3 | Featured items | API discovery (sniff) | `menu top <storeId>` | Quick view of popular items |
| 4 | Available slices | API discovery (sniff) | `menu slices` + `--json` | See today's slices across all stores |
| 5 | Delivery/pickup days | API discovery (sniff) | `scheduling time-window-days <storeId> <serviceType>` | Plan ahead |
| 6 | Time slots | API discovery (sniff) | `scheduling time-windows <storeId> <serviceType> <date>` | Pick exact window |
| 7 | Address validation | API discovery (sniff) | `addresses lookup --stdin` | Check delivery zone |
| 8 | Address autocomplete | API discovery (sniff) | `addresses autocomplete --query` | Find address quickly |
| 9 | Product pricing | API discovery (sniff) | `menu product-price --product-id` | Check prices |
| 10 | Store quotes | API discovery (sniff) | `stores quote-stores` / `stores get-quote-store <id>` | Delivery fees and minimums |
| 11 | Order history | API discovery (sniff) | `orders list` | View past orders |
| 12 | Price order | API discovery (sniff) | `orders price --stdin` | Get total before ordering |
| 13 | Submit order | API discovery (sniff) | `orders send --stdin` | Place order from CLI |
| 14 | Verify order | API discovery (sniff) | `orders verify --stdin` | Check order validity |
| 15 | Rewards card | API discovery (sniff) | `rewards reward-card` | Check points balance |
| 16 | Saved coupons | API discovery (sniff) | `rewards stored-coupons` | View saved coupons |
| 17 | Account credit | API discovery (sniff) | `rewards stored-credit` | Check credit balance |
| 18 | Order suggestions | API discovery (sniff) | `orders order-suggestion` | Get personalized picks |
| 19 | System messages | API discovery (sniff) | `system site-wide-message` | Check announcements |
| 20 | API version | API discovery (sniff) | `system version` | Check API version |
| 21 | Customer feedback | API discovery (sniff) | `feedback submit --stdin` | Send feedback |
| 22 | Auth (login/register) | API discovery (sniff) | `auth login` / `auth register` | Account management |

**Total absorbed: 22 features from sniffed API (no competing tools)**

## Transcendence

No existing tools to transcend. All features are novel for this API.

## Summary

- **Absorbed:** 0 features from existing tools (none exist)
- **Novel from sniff:** 22 features discovered via browser sniff
- **Grand total:** 22 features
- **Best existing tool:** None
- **Our advantage:** First and only CLI for the Pagliacci Pizza API
