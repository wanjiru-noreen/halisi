# Halisi

A gas refill ordering and management system that connects customers with nearby gas refill shops.

## Overview

Halisi helps users find gas refill shops, view available prices, and place refill orders. Shop owners can manage their gas prices and handle customer orders through the system.

The goal is to make gas refilling easier, faster, and more accessible.

## Problem

Many households experience challenges when refilling gas:

* Difficulty finding nearby refill shops
* Lack of price comparison between suppliers
* Time wasted making multiple calls
* No simple way to track refill requests

## Solution

Halisi provides a digital platform where:

* Customers can discover gas shops and request refills
* Shop owners can manage prices and orders
* Administrators can manage the platform

## Features

### Customer

* User registration and login
* View available gas shops
* View gas refill prices
* Place refill orders
* View order history

### Shop Owner

* Shop account management
* Update gas prices
* View incoming orders
* Accept or reject orders
* Update order status

### Admin

* Manage users
* Manage gas shops
* Monitor system activity

## Technology Stack

**Backend**

* Go (Golang)

**Frontend**

* HTML
* CSS

**Database**

* SQLite

**Architecture**

* Layered architecture:

  * Handlers
  * Services
  * Repository
  * Models

## Project Structure

```
Halisi/
│
├── cmd/
│   └── api/
│
├── internal/
│   ├── handlers/
│   ├── models/
│   ├── database/
│   ├── repository/
│   ├── services/
│   └── middleware/
│
├── web/
│   ├── templates/
│   └── static/
│
└── README.md
```

## Future Improvements

* Mobile application
* Location-based shop discovery
* Online payments
* Delivery tracking
* Customer reviews

## Author

Wanjiru Noreen