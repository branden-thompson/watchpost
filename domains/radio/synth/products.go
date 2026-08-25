// Package synth is the synthesized NOAA-Weather-Radio-style broadcast (B4
// step 2, architecture §5 "Synth", AI-13): when no relay carries a
// location's transmitter — 89 % of them — the NWS text products the real
// broadcast reads from are fetched, normalized into spoken English, and
// voiced locally. The narration never enters an argv element (§10.5).
package synth

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
)

// productOrder lists the types read on the air, in broadcast order (AI-13):
// the zone forecast is the backbone; the hazardous-weather outlook and
// short-term / special statements are read when the office issued them.
func productOrder() []string { return []string{"HWO", "SPS", "NOW", "ZFP"} }

// Product is one issued NWS text product.
type Product struct {
	ID     string
	Type   string
	Office string
	Issued time.Time
	Text   string
}

// Products fetches the latest product of each type for an office (the
// CWA id from the resolved point, e.g. "SGX"), honouring the server's
// cache lifetimes; missing types are skipped (HWO/NOW are not always issued).
type Products struct {
	client *httpx.Client
	base   string
}

// NewProducts builds the fetcher. base "" means api.weather.gov.
func NewProducts(client *httpx.Client, base string) *Products {
	if base == "" {
		base = "https://api.weather.gov"
	}
	return &Products{client: client, base: base}
}

// Latest returns the newest product per type in broadcast order.
func (p *Products) Latest(ctx context.Context, office string) ([]Product, error) {
	if err := invariant.Check(office != "", "synth: office id is required"); err != nil {
		return nil, err
	}
	var out []Product
	for _, typ := range productOrder() {
		pr, ok, err := p.latestOf(ctx, office, typ)
		if err != nil {
			return out, err
		}
		if ok {
			out = append(out, pr)
		}
	}
	return out, nil
}

func (p *Products) latestOf(ctx context.Context, office, typ string) (Product, bool, error) {
	var list struct {
		Graph []struct {
			ID           string    `json:"id"`
			IssuanceTime time.Time `json:"issuanceTime"`
		} `json:"@graph"`
	}
	if _, err := p.client.GetJSON(ctx, fmt.Sprintf("%s/products/types/%s/locations/%s", p.base, typ, office), &list, httpx.TTL(10*time.Minute)); err != nil {
		return Product{}, false, err
	}
	if len(list.Graph) == 0 {
		return Product{}, false, nil
	}
	sort.Slice(list.Graph, func(i, j int) bool { return list.Graph[i].IssuanceTime.After(list.Graph[j].IssuanceTime) })
	var doc struct {
		ID            string    `json:"id"`
		ProductCode   string    `json:"productCode"`
		IssuingOffice string    `json:"issuingOffice"`
		IssuanceTime  time.Time `json:"issuanceTime"`
		ProductText   string    `json:"productText"`
	}
	// A product's text is immutable once issued: cache it for the day.
	if _, err := p.client.GetJSON(ctx, p.base+"/products/"+list.Graph[0].ID, &doc, httpx.TTL(24*time.Hour)); err != nil {
		return Product{}, false, err
	}
	return Product{ID: doc.ID, Type: typ, Office: strings.TrimPrefix(doc.IssuingOffice, "K"), Issued: doc.IssuanceTime, Text: doc.ProductText}, true, nil
}
