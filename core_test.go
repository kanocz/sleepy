package sleepy

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"
)

type Item struct{}

func (item Item) Get(values url.Values, headers http.Header) (int, interface{}, http.Header) {
	items := []string{"item1", "item2"}
	data := map[string][]string{"items": items}
	return 200, data, nil
}

func TestBasicGet(t *testing.T) {

	var api = NewAPI()
	api.AddResource(Item{}, "/items", "/bar", "/baz")
	go func() {
		err := api.Start("localhost", 3000)
		if nil != err {
			fmt.Println("Error starting sleepy.api:", err)
		}
	}()

	// avoid errors on slow CI servers
	time.Sleep(time.Second / 10)

	resp, err := http.Get("http://localhost:3000/items")
	if err != nil {
		t.Error(err)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	if string(body) != "{\n  \"items\": [\n    \"item1\",\n    \"item2\"\n  ]\n}" {
		t.Error("Not equal.")
	}
}
