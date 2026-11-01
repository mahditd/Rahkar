package nemuda

import "testing"

func TestParseSelectsCarPriceWithAttribution(t *testing.T) {
	data := []byte(`{
      "computedAt":"2026-11-01T10:00:00Z",
      "categories":[
        {"category":"motorcycle","brands":[{"slug":"moto","dataPoints":1000,"models":[{"slug":"bike","nameFa":"موتور","dataPoints":1000,"years":[{"modelYear":1404,"dataPoints":100,"price":999}]}]}]},
        {"category":"car","brands":[{"slug":"peugeot","dataPoints":100,"models":[{"slug":"207i","nameFa":"پژو 207i","dataPoints":100,"years":[{"modelYear":1403,"dataPoints":12,"price":1000},{"modelYear":1404,"dataPoints":70,"price":2000000000,"mileageKm":25000}]}]}]}
      ]
    }`)
	got, err := parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].PriceToman != 2000000000 || got.Items[0].SourceURL != "https://nemuda.com/car/peugeot/207i/1404" {
		t.Fatalf("unexpected market overview: %+v", got)
	}
}
