package utils

import (
	"sort"
	"strings"

	"airbnb-scraper-w3e/models"
)

type CityCount struct {
	City  string
	Count int
}

type SummaryStats struct {
	TotalListings         int
	AveragePrice          float32
	MinimumPrice          float32
	MaximumPrice          float32
	MostExpensiveProperty models.Listing
	ListingsPerCity       []CityCount
	TopRatedProperties    []models.Listing
}

func BuildSummaryStats(results []models.CityResult) SummaryStats {
	all := make([]models.Listing, 0)
	cityCounts := make(map[string]int)

	for _, result := range results {
		if result.Err != nil {
			continue
		}
		city := strings.TrimSpace(result.City)
		if city == "" {
			city = "Unknown"
		}
		for _, listing := range result.Listings {
			all = append(all, listing)
			cityCounts[city]++
		}
	}

	stats := SummaryStats{TotalListings: len(all)}
	if len(all) == 0 {
		return stats
	}

	minPrice := all[0].Price
	maxPrice := all[0].Price
	mostExpensive := all[0]
	var totalPrice float32

	for _, listing := range all {
		totalPrice += listing.Price
		if listing.Price < minPrice {
			minPrice = listing.Price
		}
		if listing.Price > maxPrice {
			maxPrice = listing.Price
			mostExpensive = listing
		}
	}

	stats.AveragePrice = totalPrice / float32(len(all))
	stats.MinimumPrice = minPrice
	stats.MaximumPrice = maxPrice
	stats.MostExpensiveProperty = mostExpensive

	perCity := make([]CityCount, 0, len(cityCounts))
	for city, count := range cityCounts {
		perCity = append(perCity, CityCount{City: city, Count: count})
	}
	sort.Slice(perCity, func(i, j int) bool {
		if perCity[i].Count == perCity[j].Count {
			return perCity[i].City < perCity[j].City
		}
		return perCity[i].Count > perCity[j].Count
	})
	stats.ListingsPerCity = perCity

	topRated := make([]models.Listing, len(all))
	copy(topRated, all)
	sort.Slice(topRated, func(i, j int) bool {
		if topRated[i].Rating == topRated[j].Rating {
			return topRated[i].Price > topRated[j].Price
		}
		return topRated[i].Rating > topRated[j].Rating
	})
	if len(topRated) > 5 {
		topRated = topRated[:5]
	}
	stats.TopRatedProperties = topRated

	return stats
}
