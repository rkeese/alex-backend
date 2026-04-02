# Requirements for the finance sector

## Financial year (Wirtschaftsjahr)

We always consider finances for a financial year, and a financial year corresponds to a calendar year.
This means that we must always be able to look at the finances of previous years.

## Annual accounts (Jahresabschluss)

At the end of each year, annual financial statements should always be prepared.

## Barges (Barkasse)

Cash entries are assigned to a respective booking account.
We require a receipt for every cash transaction.
A receipt must contain the following information: date, note and recipient or payer.

## Bank accounts (Bankkonten)

It must also be possible to include bookings from different bank accounts.
To record account movements, it should be possible to import Excel or CSV files containing the transactions.

## Entries (Buchungen)

It must be possible to assign cash or account entries to specific entry accounts (SKR49).
The entry accounts relevant to an club must be managed in a separate table.

## Already implemented database tables

In context with the above defined requirements the database contain already following tables:
booking_accounts, bookings and club_bank_accounts.

## Common requirements

All implementations must controllable over the rest interface.
Every change at the rest endpoints must be documented within the API.md file.
Additionaly a description text for the frontend implementation will very helpful.
