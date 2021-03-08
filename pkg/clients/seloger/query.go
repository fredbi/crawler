package seloger

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var debug = os.Getenv("DEBUG_TEST") != ""

// Credentials for this API
type Credentials struct {
	Token    string
	Datadome string
}

// Listing describe a classified
type Listing struct {
	Raw   []byte
	AsMap map[string]interface{}
}

const (
	scheme         = "https"
	connectHost    = "myspace-slwcf.svc.groupe-seloger.com"
	connectEnpoint = "/AuthentificationService.svc/ConnecterUtilisateur"
	origin         = "https://wwww.seloger.com"
	userAgent      = `Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:86.0) Gecko/20100101 Firefox/86.0`
	connectBody    = `{"request":{"Email":"%s","MotDePasse":"%s"}}`
)

func setCommonHeaders(req *http.Request) {
	req.Header.Add("Host", connectHost)
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Origin", origin)
	req.Header.Add("Referer", origin+"/")
	req.Header.Add("User-Agent", userAgent)
}

func logReq(req *http.Request) {
	if !debug {
		return
	}
	dump, _ := httputil.DumpRequestOut(req, true)
	log.Printf("DEBUG: request: %s", string(dump))
}

func logResp(resp *http.Response) {
	if !debug {
		return
	}
	dump, _ := httputil.DumpResponse(resp, true)
	log.Printf("DEBUG: response: %s", string(dump))
}

// Connect to the service
func Connect(user, password string) (*Credentials, error) {
	client := &http.Client{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connectURL := &url.URL{
		Scheme: scheme,
		Host:   connectHost,
		Path:   connectEnpoint,
	}

	reqBody := strings.NewReader(fmt.Sprintf(connectBody, user, password))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, connectURL.String(), reqBody)
	if err != nil {
		return nil, err
	}

	setCommonHeaders(req)
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	logReq(req)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	logResp(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d [%s]", resp.StatusCode, resp.Status)
	}

	res := new(Credentials)
	cookies := resp.Cookies()
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		switch cookie.Name {

		case "Token":
			res.Token = cookie.Value
		case "Datadome":
			res.Datadome = cookie.Value
		}
	}

	if res.Token == "" {
		return nil, fmt.Errorf("could not acquire auth cookie")
	}

	return res, nil
}

const (
	listingHost     = `www.seloger.com`
	listingEndpoint = "/list.htm"
)

// GetListings retrieves a listing of classified
func GetListings(creds *Credentials) ([]Listing, error) {

	//?tri=initial&enterprise=0&idtypebien=2,14,13,9,1&pxMax=800000&idtt=2,5&naturebien=1,2,4&ci=920040&m=search_hp_new
	client := &http.Client{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	values := make(url.Values, 5)
	/*
		values.Add("tri", "initial")
		values.Add("enterprise", "0")
		values.Add("pxMax", "800000")
		values.Add("ci", "920040")
		values.Add("idtt", "2,5")
		values.Add("m", "search_hp_new") // search_refine
		values.Add("idtypebien", "2,14,13,9")
		values.Add("naturebien", "1,2,4")
	*/
	values.Add("enterprise", "0")
	values.Add("projects", "2,5")
	values.Add("types", "1,2,9,13")
	values.Add("natures", "1,2")
	//values.Add("places", `[{"inseeCodes":[920040]}]`)
	values.Add("places", `[{"inseeCodes":[890024]}]`)
	values.Add("price", "0/800000")
	values.Add("qsVersion", "1")
	values.Add("m", "search_refine")
	values.Add("groundsurface", "50/100")
	values.Add("surface", "100/500")
	values.Add("proximities", "0,10")
	values.Add("rooms", "4")
	values.Add("bedrooms", "2")
	values.Add("mandatorycommodities", "1")
	//projects=2,5&types=1,2,9,13,14&natures=1,2,4&places=[{"inseeCodes":[920040]}]&price=NaN/800000&enterprise=0&qsVersion=1.0&m=search_refine
	//https://www.seloger.com/list.htm?
	//projects=2,5&types=1,2,9,13,14&natures=1,2,4&places=[{"inseeCodes":[890024]}]
	// &proximities=0,10&price=NaN/800000&groundsurface=50/100&surface=100/500&bedrooms=2&rooms=3&mandatorycommodities=1&enterprise=0&qsVersion=1.0&m=search_refine

	listURL := &url.URL{
		Scheme:   scheme,
		Host:     listingHost,
		Path:     listingEndpoint,
		RawQuery: values.Encode(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL.String(), nil)
	if err != nil {
		return nil, err
	}

	setCommonHeaders(req)

	// TODO: pagination
	cookie := &http.Cookie{
		Name:  "ep-authorization",
		Value: creds.Token,
	}
	req.Header.Add("Cookie", cookie.String())

	logReq(req)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	logResp(resp)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d [%s]", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	doc.Find(`a[name*="classified-link"]`).Each(func(i int, s *goquery.Selection) {
		//band := s.Find("a").Text()
		//title := s.Find("i").Text()
		href, _ := s.Attr("href")
		u, _ := url.Parse(href)
		u.Fragment = ""
		u.RawQuery = ""
		u.Path = path.Join("annonces", path.Base(u.Path))
		log.Printf("anchor %d: %s", i, u.String())
	})

	listings := make([]Listing, 0, 100)

	/*
		listings = append(listings, Listing{
			Raw: doc.,
		})
	*/

	return listings, nil
}

const (
	/*
		connect
		POST
		https://myspace-slwcf.svc.groupe-seloger.com/AuthentificationService.svc/ConnecterUtilisateur
		body:
		{"request":{"Email":"frederic.bidon@yahoo.com","MotDePasse":"B6cAjvaC65TG28V"}}

		response header:
		Token=5BolETJXuupDH6ABX8QJ55kDcivswPtHceHwRKzuZGcO7gHpMxti9pUtzJRUhvpve4HKemALrDqvv03KbtsILZg0GL0ZSr5nNQf26WsdPxSSO3umXu1Gq9ZdNLp0opGiNhL94eK05tFQHKzO7TAFcME549v_GIruqbpnyqAoD6M1;path=/;expires=06/03/2031 08:23:08;Max-Age=315360000
	*/

	url1  = `https://myspace-slwcf.svc.groupe-seloger.com/AnnonceAlertePrixService.svc/RecupererAnnoncesIds?request={"Token":"5BolETJXuupDH6ABX8QJ55tIS8Kxd_dDk5ads4lvnweUZ4h-8IoxdmhtFUG7rI8ttOO_km7QsTUhNffrpZ--sIM-v7cfnzOOclFIvqnaCxqReHWAduEHC1YVIjpfZtsYX68SJmZ8YaxH8QF8Ajk2J06TSjuWy16uBow5HPra-nQ1"}`
	token = `5BolETJXuupDH6ABX8QJ55tIS8Kxd_dDk5ads4lvnweUZ4h-8IoxdmhtFUG7rI8ttOO_km7QsTUhNffrpZ--sIM-v7cfnzOOclFIvqnaCxqReHWAduEHC1YVIjpfZtsYX68SJmZ8YaxH8QF8Ajk2J06TSjuWy16uBow5HPra-nQ1`
	/*
		headers:
		authorization: "token"
		api-supported-version: 3.5
	*/
	url2 = `/api/3.5/bigdata/users/keys/hiddenlist`
	url3 = `/api/3.5/bigdata/users/keys/viewedlsts`
	url4 = `/api/3.5/bigdata/users/keys/lastsearch`
	/*
		  search
			POST
			lastsearch: {
				enterprise: bool
				natures: [ 1, 2, 4 ],
				places: [
				inseeCodes:  [ 920040 ]
				label: "Issy les Moulineaux"
				],
				price: {
					max: 600000,
					min:  null
				},
				projects: [ 2, 5 ],
				types: [ 1, 2 ]
			}
	*/
	url41 = `/api/3.5/bigdata/users/keys`
	url42 = `/api/3.5/listings`
	/*
		autocomplete.svc.groupe-seloger.com

		query params
		key: ci
		value: 920040
	*/
	url5 = `/api/v2.0/auto/resolve/fra/1`
	/*
		www.seloger.com
		POST

		cookies in request:

		datadome:

		didomi_token:
		eyJ1c2VyX2lkIjoiMTc4MDY1NjAtMDllZi02NWRkLWE4MGEtZDdlZmZiZGEwZmVkIiwiY3JlYXRlZCI6IjIwMjEtMDMtMDZUMDc6MDA6MDYuMzQ1WiIsInVwZGF0ZWQiOiIyMDIxLTAzLTA2VDA3OjAwOjA2LjM0NVoiLCJ2ZW5kb3JzIjp7ImVuYWJsZWQiOlsiZmFjZWJvb2siLCJnb29nbGUiLCJjOm9tbml0dXJlLWFkb2JlLWFuYWx5dGljcyIsImM6aGFydmVzdC1QVlRUdFVQOCIsImM6ZmFjZWJvb2stYnRDNFpXNnIiXX0sInB1cnBvc2VzIjp7ImVuYWJsZWQiOlsiYW5hbHlzZWRlLVZEVFVVaG42Iiwic29jaWFsIiwiZGV2aWNlX2NoYXJhY3RlcmlzdGljcyIsImdlb2xvY2F0aW9uX2RhdGEiXX0sInZlbmRvcnNfbGkiOnsiZW5hYmxlZCI6WyJnb29nbGUiXX0sInZlcnNpb24iOjIsImFjIjoiQWt1QUNBa3MuQWt1QUNBa3MifQ==

		{
			"user_id":"17806560-09ef-65dd-a80a-d7effbda0fed",
			"created":"2021-03-06T07:00:06.345Z",
			"updated":"2021-03-06T07:00:06.345Z",
			"vendors":{"enabled":["facebook","google","c:omniture-adobe-analytics","c:harvest-PVTTtUP8","c:facebook-btC4ZW6r"]},
			"purposes":{
				"enabled":["analysede-VDTUUhn6","social","device_characteristics","geolocation_data"]
			},
			"vendors_li":{"enabled":["google"]},
			"version":2,
			"ac":"AkuACAks.AkuACAks"
		}

		ep-authorization: (another token: encrypted)
		5BolETJXuupDH6ABX8QJ5whmH8FnRTAd4ZNZIHZRkbeAtYpxY7q5GOQWZhdnbRNLPi4BTUVCuGhLJEqaZLadyErVkpS3xr38dZqNAtuAE23PqWwUTNSRUQYvg3cXm-S16SVPUnLJfIK901XsQHE7Ffc7bKHYR_qBXHSVwAXw47s1


		eu-consent-v2:
		CPCn5SfPCn5SfAHABBENBPCsAP_AAH_AAAAAHetf_X_fb3_j-_59_9t0eY1f9_7_v-0zjhfds-8Nyf_X_L8X42M7vF36pq4KuR4Eu3LBIQdlHOHcTUmw6okVrTPsbk2Mr7NKJ7PEinMbe2dYGH9_n93TuZKY7__s___z__-__v__7_f_r-3_3_vp9X---_e_V399xKB3QBJhqXwEWYljgSTRpVCiBCFcSHQCgAooRhaJrCAlcFOyuAj1BAwAQGoCMCIEGIKMWAQAAAQBJREBIAeCARAEQCAAEAKkBCAAjYBBYAWBgEAAoBoWIEUAQgSEGRwVHKYEBEi0UE8lYAlF3sYYQhlFgBQKP6KjARKEAAAA.f_gAD_gAAAAA


		visitId: ##


	*/
	url6 = `/list/christie/serialize`
	/*
		POST

		lastsearch above
		result : {
			nb: 384
			nbgeoloc: 0
		}
	*/
	url7 = `/list/christie/count`
	/*
		listing html
		cookies above
		GET

		query params:
		tri: "initial"
		enterprise: 0
		idtypebien: 2,1
		pxMax: 600000
		idtt: 2,5
		naturebien: 1,2,4
		ci: 920040
		m: "search_hp_new"

		response page html avec le listing
	*/
	url8 = `/list.htm`
)

/*
/AnnonceAlertePrixService.svc/RecupererAnnoncesIds
cors: yes
host: myspace-slwcf.svc.groupe-seloger.com
origin: https://www.seloger.com
header param:
request {
	Token: string
}
// Address: ip
// microsoft iis/10.0
// cookies: no
// Response:
// graphql
*/
func get() {
}

/*
 endpoints
 myspace-sl.svc.groupe-seloger.com
 basePath: /api/3.5/bigdata

 /users/keys/{firstname|email|lastsearch|viewedlsts|hiddenlist|homephone}
 /listings
 /listings/comments
 /projects/sharing



 www.seloger.com
 POST /list/christie/jacqueline
 POST /list/christie/serialize
 POST /list/christie/count

 www.seloger.com
 /annonces/achat/appartement/issy-les-moulineaux-92/jules-guesde-montesy/168670255.htm?projects=2,5&types=1,2&natures=1,2,4&places=[{"inseeCodes":[920040]}]&price=NaN/600000&enterprise=0&qsVersion=1.0&m=search_to_detail&Classified-ViewId=168670255
 /annonces/168670255.htm
*/
