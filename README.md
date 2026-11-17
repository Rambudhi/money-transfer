# Money Transfer Service

Money Transfer Service is a backend application built with Go for handling merchant money transfers.

The application uses asynchronous processing with RabbitMQ. When a transfer request is received, the merchant balance is deducted and the transaction is saved with `PENDING` status. The transfer is then processed by a background worker using a mock provider.

If the transfer succeeds, the transaction status is updated to `SUCCESS`. If it fails, the deducted amount is returned to the merchant balance and the transaction status is updated to `FAILED`.

## Tech Stack

- Go
- Echo Framework
- PostgreSQL
- RabbitMQ
- Docker & Docker Compose
- Uber Fx
- pgx

## How It Works

The transfer process consists of two parts.

**1. Transfer API**

When a client sends a transfer request:

1. Validate the request.
2. Check the merchant balance.
3. Deduct the transfer amount from the merchant balance.
4. Create a transaction with `PENDING` status.
5. Save an event to the `message_outbox` table.
6. Commit all database changes in one transaction.

**2. Background Worker**

The outbox publisher reads pending events and publishes them to RabbitMQ.

The worker consumes the messages and processes the transfer using the mock provider.

- If successful, the transaction is marked as `SUCCESS`.
- If failed, the merchant balance is refunded and the transaction is marked as `FAILED`.

## Architecture

```text
Client
  |
  v
Transfer API
  |
  v
PostgreSQL
  |
  |-- Deduct Merchant Balance
  |-- Create Transaction
  |-- Create Outbox Event
  |
  v
Outbox Publisher
  |
  v
RabbitMQ
  |
  v
Transfer Worker
  |
  v
Mock Provider
  |
  +-------------------+
  |                   |
  v                   v
SUCCESS              FAILED
  |                   |
  v                   v
Update Status       Refund Balance
                      |
                      v
                  Update Status
```

## Transactional Outbox

The application uses the Transactional Outbox Pattern to make sure that the transaction record and its corresponding event are saved together.

Both operations are executed in the same PostgreSQL transaction.

The outbox publisher is responsible for publishing pending events to RabbitMQ. After a successful publish, the outbox record is marked as `PUBLISHED`.

This helps prevent a situation where a transfer is saved in the database but its message is not published.

Message delivery can happen more than once, so the worker also needs to handle duplicate messages safely.

## Database Tables

The application uses three main tables:

| Table | Description |
|---|---|
| `merchants` | Stores merchant information and balance |
| `transactions` | Stores transfer transaction records |
| `message_outbox` | Stores events waiting to be published |

Transaction statuses:

- `PENDING`
- `SUCCESS`
- `FAILED`

## API

### POST /transfer

Initiates a new transfer.

Request:

```json
{
  "merchant_id": 1,
  "amount": 10000,
  "destination_account": "1234567890"
}
```

| Field | Type | Description |
|---|---|---|
| merchant_id | integer | Merchant ID |
| amount | integer | Transfer amount |
| destination_account | string | Destination account number |

The API creates the transfer request and returns without waiting for the mock provider to finish processing it.

## Requirements

Make sure you have installed:

- Go
- Docker
- Docker Compose

## Configuration

Copy the example environment file:

```bash
cp .env.example .env
```

Update the environment variables according to your local configuration.

## Running the Application

Start all services using Docker Compose:

```bash
docker compose up -d --build
```

Check running containers:

```bash
docker compose ps
```

Check API logs:

```bash
docker compose logs -f api
```

Check worker logs:

```bash
docker compose logs -f worker
```

Stop all services:

```bash
docker compose down
```

## Testing

You can test the transfer endpoint using Postman or cURL.

```bash
curl -X POST http://localhost:8080/transfer \
  -H "Content-Type: application/json" \
  -d '{
    "merchant_id": 1,
    "amount": 10000,
    "destination_account": "1234567890"
  }'
```

Check the latest transactions:

```sql
SELECT *
FROM transactions
ORDER BY created_at DESC;
```

Check merchant balances:

```sql
SELECT *
FROM merchants;
```

Check outbox messages:

```sql
SELECT *
FROM message_outbox
ORDER BY id DESC;
```

## Load Testing

A load testing script is available in:

```text
scripts/load_test.sh
```

Run it using:

```bash
bash scripts/load_test.sh
```

The script sends concurrent transfer requests and reports the request count, successful responses, rejected requests, server errors, network errors, and actual RPS.

## Notes

- The transfer provider is mocked and does not connect to a real banking or payment provider.
- Transfer processing runs asynchronously through RabbitMQ.
- Merchant balance deduction and outbox event creation are committed together.
- Failed transfers should refund the deducted amount.
- RabbitMQ message processing should be idempotent to prevent duplicate processing and refunds.

## Author

Developed for the Doitpay Backend Engineer Technical Assessment.