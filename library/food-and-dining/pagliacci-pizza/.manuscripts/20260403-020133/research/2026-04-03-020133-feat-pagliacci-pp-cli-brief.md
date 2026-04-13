# Pagliacci CLI Brief

## API Identity
- Domain: Local pizza delivery & carryout (Seattle area)
- Users: Seattle-area pizza enthusiasts, office lunch organizers, anyone who orders from Pagliacci regularly
- Data profile: Menu (pizzas, salads, sides, drinks, specials), Locations (stores, hours, delivery zones), Orders (cart, checkout, history), Rewards (loyalty points, earned rewards)
- API type: Undocumented — Angular SPA with backend API. No public spec, no community wrappers.
- Auth: Browser session cookies for account features (rewards, order history, saved addresses)

## Reachability Risk
- **Unknown** — No community wrappers to check. Pagliacci is a small regional chain; their API may have bot detection or rate limiting. Will probe during sniff.

## Top Workflows
1. **Order a pizza for delivery** — Browse menu → customize → add to cart → checkout → track
2. **Browse the menu and specials** — See what's available, check seasonal specials, view prices
3. **Find nearest location and hours** — Store locator, delivery zone check, hours
4. **Check rewards balance** — Loyalty points, available rewards, redemption
5. **Reorder a favorite** — Past orders → one-click reorder

## Table Stakes
- Menu browsing with categories (pizza, salads, sides, drinks)
- Pizza customization (size, toppings, crust)
- Location finder with delivery zone check
- Order placement and tracking
- Rewards/loyalty program access

## Data Layer
- Primary entities: MenuItems, Locations, Orders, Rewards
- Sync cursor: Menu changes with seasonal specials (weekly cache OK)
- FTS/search: Menu item search by name/description, location search by address

## Product Thesis
- Name: pagliacci-pp-cli
- Why it should exist: Order your favorite Seattle pizza from the terminal. No existing CLI or wrapper exists for Pagliacci — this would be the first programmatic interface to their ordering system. Useful for office lunch automation, reward tracking, and the nerd cred of ordering artisan pizza from the command line.

## Build Priorities
1. Menu browsing + location finder (foundation — needs sniff to discover endpoints)
2. Order workflow: build cart → customize → checkout (core value)
3. Rewards/loyalty tracking (account features)
4. Order history and reorder (power user)
