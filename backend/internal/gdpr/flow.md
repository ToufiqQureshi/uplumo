# Account deletion and feedback flow

Account menu or Personal settings → confirmation → `POST /api/auth/delete-account` → signed session and origin check → one Postgres transaction → clear browser auth cookies and return to the homepage. The deletion calls zero paid providers and uses zero credits.
