# Shopping Cart

A micro-service to store and manipluate shopping carts and their items.

## How to Run

### Migrations

    goose -dir ./migrations sqlite3 "file:./testdata/db.sqlite3" up

### Local

    go run . -help
    go run .

### Docker

    docker build -t shoppingcart:latest .
    docker run --rm -p5000:5000 shoppingcart:latest

## REST API

### Cart

#### Create

    curl localhost:5000/v1/cart --user Aladdin:OpenSesame -v -d'{"user_id":100,"line_items":[{"product_id":20,"quantity":50}]}'

#### Show

    curl -v --user Aladdin:OpenSesame localhost:5000/v1/cart/1

#### Empty

    curl -v --user Aladdin:OpenSesame localhost:5000/v1/cart/1 -XDELETE

### Line Items

#### Add

    curl -v --user Aladdin:OpenSesame localhost:5000/v1/cart/1/item -d'[{"product_id":20,"quantity":5},{"product_id":99,"quantity":10}]' -XPUT

#### Remove

    curl -v --user Aladdin:OpenSesame localhost:5000/v1/cart/1/item/1 -XDELETE

## Missing Bits

- [ ] Integration tests
- [ ] Proper Authentication Middleware
- [ ] ...