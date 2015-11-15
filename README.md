## Sleepy

#### A RESTful framework for Go

Sleepy is a micro-framework for building RESTful APIs.

```go
package main

import (
    "net/url"
    "net/http"
    "github.com/kanocz/sleepy"
    "github.com/julienschmidt/httprouter"
)

type Item struct { }

func (item Item) Get(values url.Values, headers http.Header, params httprouter.Params) (int, interface{}, http.Header) {
    items := []string{"item1", "item2"}
    data := map[string][]string{"items": items, "name": params.ByName("name")}
    return 200, data, http.Header{"Content-type": {"application/json"}}
}

func main() {
    item := new(Item)

    api := sleepy.NewAPI()
    api.AddResource(item, "/items/:name")
    api.Start(3000)
}
```

Now if we curl that endpoint:

```bash
$ curl localhost:3000/items/hello
{"items": ["item1", "item2"], "name": "hello"}
```

`sleepy` has not been officially released yet, as it is still in active
development.

## Docs

Original documentation lives [here](http://godoc.org/github.com/dougblack/sleepy).

## License

`sleepy` is released under the [MIT License](http://opensource.org/licenses/MIT).
