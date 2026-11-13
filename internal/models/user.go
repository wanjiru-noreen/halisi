package models

type Role string

const (
	Customer      Role = "customer"
	Vendor        Role = "vendor"
	Rider         Role = "rider"
	Administrator Role = "admin"
)

type Permission string

const (
	CustomerResources Permission = "customer.resources"
	VendorResources   Permission = "vendor.resources"
	RiderAssignments  Permission = "rider.assignments"
	Administration    Permission = "administration"
)

// Unknown roles and permissions are denied. Admin privileges are explicit,
// not a bypass for another role's ownership-scoped endpoints.
func Allows(role Role, permission Permission) bool {
	switch permission {
	case CustomerResources:
		return role == Customer
	case VendorResources:
		return role == Vendor
	case RiderAssignments:
		return role == Rider
	case Administration:
		return role == Administrator
	default:
		return false
	}
}

func DashboardPath(role Role) (string, bool) {
	switch role {
	case Customer:
		return "/customer/dashboard", true
	case Vendor:
		return "/vendor/dashboard", true
	case Rider:
		return "/rider/dashboard", true
	case Administrator:
		return "/admin/dashboard", true
	default:
		return "", false
	}
}
