package http

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/akr24/sloan-marketplace/backend/internal/domain"
	"github.com/akr24/sloan-marketplace/backend/internal/usecase"
)

type listingsHandler struct {
	listings *usecase.Listings
}

// createListingRequest takes the price in dollars, as people type it.
type createListingRequest struct {
	EventName string  `json:"event_name"`
	EventDate string  `json:"event_date"` // YYYY-MM-DD
	Venue     string  `json:"venue"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
	Notes     string  `json:"notes"`
}

type setStatusRequest struct {
	Status domain.Status `json:"status"`
}

func (h listingsHandler) create(c echo.Context) error {
	var req createListingRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "malformed JSON")
	}
	date, err := time.Parse(time.DateOnly, req.EventDate)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "event_date must be YYYY-MM-DD")
	}
	l, err := h.listings.Create(c.Request().Context(), currentUserID(c), domain.SourceWeb, usecase.ListingInput{
		EventName:  req.EventName,
		EventDate:  date,
		Venue:      req.Venue,
		Quantity:   req.Quantity,
		PriceCents: dollarsToCents(req.Price),
		Notes:      req.Notes,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, l)
}

// search reads filters from the query string:
// q, date_from, date_to (YYYY-MM-DD), min_price, max_price (dollars),
// min_qty, sort (soonest|newest|price_low|price_high), limit.
func (h listingsHandler) search(c echo.Context) error {
	f := domain.ListingFilter{
		Query: c.QueryParam("q"),
		Sort:  domain.ListingSort(c.QueryParam("sort")),
	}
	var err error
	if f.DateFrom, err = queryDate(c, "date_from"); err != nil {
		return err
	}
	if f.DateTo, err = queryDate(c, "date_to"); err != nil {
		return err
	}
	if f.MinPriceCents, err = queryDollars(c, "min_price"); err != nil {
		return err
	}
	if f.MaxPriceCents, err = queryDollars(c, "max_price"); err != nil {
		return err
	}
	if f.MinQuantity, err = queryInt(c, "min_qty"); err != nil {
		return err
	}
	if f.Limit, err = queryInt(c, "limit"); err != nil {
		return err
	}

	list, err := h.listings.Search(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, list)
}

func (h listingsHandler) get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	}
	l, err := h.listings.Get(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, l)
}

func (h listingsHandler) mine(c echo.Context) error {
	list, err := h.listings.Mine(c.Request().Context(), currentUserID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, list)
}

func (h listingsHandler) setStatus(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	}
	var req setStatusRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "malformed JSON")
	}
	l, err := h.listings.SetStatus(c.Request().Context(), currentUserID(c), id, req.Status)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, l)
}

func dollarsToCents(d float64) int {
	return int(math.Round(d * 100))
}

func queryDate(c echo.Context, name string) (*time.Time, error) {
	v := c.QueryParam(name)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.DateOnly, v)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, name+" must be YYYY-MM-DD")
	}
	return &t, nil
}

func queryDollars(c echo.Context, name string) (*int, error) {
	v := c.QueryParam(name)
	if v == "" {
		return nil, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, name+" must be a dollar amount")
	}
	cents := dollarsToCents(f)
	return &cents, nil
}

func queryInt(c echo.Context, name string) (int, error) {
	v := c.QueryParam(name)
	if v == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, echo.NewHTTPError(http.StatusBadRequest, name+" must be a whole number")
	}
	return n, nil
}
