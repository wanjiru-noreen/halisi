package templdata

import "halisi/internal/models"

// ShopsPage holds data for the shops listing page
type ShopsPage struct {
	Shops []models.Shop
}

// ShopPage holds data for the single shop detail page
type ShopPage struct {
	Shop models.Shop
}

// OrderPage holds data for the order form page
type OrderPage struct {
	Shops         []models.Shop
	SelectedShopID int
}

// OrdersPage holds data for the orders listing page
type OrdersPage struct {
	Orders []models.Order
}

// OrderConfirmationPage holds data for the order confirmation page
type OrderConfirmationPage struct {
	Order models.Order
}

// AdminPage holds data for the admin dashboard page
type AdminPage struct {
	TotalUsers  int
	TotalShops  int
	TotalOrders int
	Users       []models.User
	Shops       []models.Shop
	Orders      []models.Order
}

// ShopOrdersPage holds data for the shop owner's orders page
type ShopOrdersPage struct {
	Shop   models.Shop
	Orders []models.Order
	Role   string
}

// ShopManagePage holds data for the shop owner's manage page
type ShopManagePage struct {
	Shop models.Shop
	Role string
}
