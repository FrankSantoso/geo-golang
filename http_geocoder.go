package geo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/corpix/uarand"
	"github.com/valyala/fasthttp"
	"net/url"
	"strconv"
	"time"
)

// DefaultTimeout for the request execution
const DefaultTimeout = time.Second * 8

// ErrTimeout occurs when no response returned within timeoutInSeconds
var ErrTimeout = errors.New("TIMEOUT")

var client = &fasthttp.Client{
	NoDefaultUserAgentHeader:      true,
	MaxConnsPerHost:               1000000,
	ReadBufferSize:                8192,
	WriteBufferSize:               8192,
	ReadTimeout:                   DefaultTimeout,
	WriteTimeout:                  DefaultTimeout,
	MaxIdleConnDuration:           2 * time.Minute,
	DisableHeaderNamesNormalizing: true,
}

// EndpointBuilder defines functions that build urls for geocode/reverse geocode
type EndpointBuilder interface {
	GeocodeURL(string) string
	ReverseGeocodeURL(Location) string
}

// ResponseParserFactory creates a new ResponseParser
type ResponseParserFactory func() ResponseParser

// ResponseParser defines functions that parse response of geocode/reverse geocode
type ResponseParser interface {
	Location() (*Location, error)
	Address() (*Address, error)
}

// HTTPGeocoder has EndpointBuilder and ResponseParser
type HTTPGeocoder struct {
	EndpointBuilder
	ResponseParserFactory
}

// Geocode returns location for address
func (g HTTPGeocoder) Geocode(address string) (*Location, error) {
	responseParser := g.ResponseParserFactory()

	ctx, cancel := context.WithTimeout(context.TODO(), DefaultTimeout)
	defer cancel()

	type geoResp struct {
		l *Location
		e error
	}
	ch := make(chan geoResp, 1)

	go func(ch chan geoResp) {
		if err := response(ctx, g.GeocodeURL(url.QueryEscape(address)), responseParser); err != nil {
			ch <- geoResp{
				l: nil,
				e: err,
			}
		}

		loc, err := responseParser.Location()
		ch <- geoResp{
			l: loc,
			e: err,
		}
	}(ch)

	select {
	case <-ctx.Done():
		return nil, ErrTimeout
	case res := <-ch:
		return res.l, res.e
	}
}

// ReverseGeocode returns address for location
func (g HTTPGeocoder) ReverseGeocode(lat, lng float64) (*Address, error) {
	responseParser := g.ResponseParserFactory()

	ctx, cancel := context.WithTimeout(context.TODO(), DefaultTimeout)
	defer cancel()

	type revResp struct {
		a *Address
		e error
	}
	ch := make(chan revResp, 1)

	go func(ch chan revResp) {
		if err := response(ctx, g.ReverseGeocodeURL(Location{lat, lng}), responseParser); err != nil {
			ch <- revResp{
				a: nil,
				e: err,
			}
		}

		addr, err := responseParser.Address()
		ch <- revResp{
			a: addr,
			e: err,
		}
	}(ch)

	select {
	case <-ctx.Done():
		return nil, ErrTimeout
	case res := <-ch:
		return res.a, res.e
	}
}

type commonRequest struct {
	url string
	obj ResponseParser
}

// Response gets response from url
func response(ctx context.Context, url string, obj ResponseParser) error {
	comm := &commonRequest{
		url,
		obj,
	}
	errChan := make(chan error, 1)
	go func() {
		errChan <- fetch(comm)
	}()
	if err := <-errChan; err != nil {
		return err
	}
	return nil
}

func fetch(c *commonRequest) error {
	retries := 0
	res := fasthttp.AcquireResponse()
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetMethod("GET")
	req.Header.SetUserAgent(uarand.GetRandom())
	req.SetRequestURI(c.url)
	if err := fasthttp.DoTimeout(req, res, client.ReadTimeout); err != nil {
		retries++
		Logger.Printf("Error fetching response, retrying...")
		if retries < 3 {
			handleRetries(err, c)
		}
		return err
	}

	bods := bytes.Trim(res.Body(), " []")
	if err := json.Unmarshal(bods, c.obj); err != nil {
		return err
	}
	fasthttp.ReleaseResponse(res)
	return nil
}

func handleRetries(err error, c *commonRequest) {
	if err == fasthttp.ErrTimeout || err == fasthttp.ErrDialTimeout {
		client.ReadTimeout += 5 * time.Millisecond
		fetch(c)
	} else {
		return
	}
}

// ParseFloat is a helper to parse a string to a float
func ParseFloat(value string) float64 {
	f, _ := strconv.ParseFloat(value, 64)
	return f
}
