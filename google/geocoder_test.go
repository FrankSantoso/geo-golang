package google_test

import (
	"github.com/codingsince1985/geo-golang"
	"github.com/codingsince1985/geo-golang/google"
	"testing"
)

func TestLocation(t *testing.T) {
	location := google.Geocoder.Geocode("Melbourne VIC")
	if location.Lat != -37.814107 || location.Lng != 144.96328 {
		t.Error("Geocode() failed", location)
	}
}

func TestAddress(t *testing.T) {
	address := google.Geocoder.ReverseGeocode(geo.Location{-37.814106, 144.963287})
	if address != "Melbourne's GPO, 250 Bourke Street Mall, Melbourne VIC 3000, Australia" {
		t.Error("ReverseGeocode() failed", address)
	}
}