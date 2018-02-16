# Zevenet Loadbalancer for Go

This library provides access to the Zevenet loadbalancer using its ZAPI (REST API) of v3.1.

It is tested using Zevenet Community Edition 5.0 but should probably work with the Enterprise Edition, too.

**[Click here to open the GoDoc documentation.](https://godoc.org/github.com/konsorten/zevenet-lb-go)**

## Usage

To use the API, simply create a new session:

```go
import zevenet "github.com/konsorten/zevenet-lb-go"

func main() {
    session, _ := zevenet.Connect("myloadbalancer:444", "zapi-key", nil)

    version, _ := session.GetSystemVersion()
}
```

## ZAPI Key

The API key for the Zevenet CE API can be retrieved using the web interface:

https://myloadbalancer:444/#/system/users/zapi

If the key is empty, click the *Generate Random Key* button and *Apply*.

## Authors

The library is sponsored by the [marvin + konsorten GmbH](http://www.konsorten.de).

It is loosely based on the [go-bigip package](https://github.com/scottdware/go-bigip) by Scott Ware (and others).

We thank all the authors who provided code to this library:

* Felix Kollmann

## License

(The MIT License)

Copyright (c) 2018 marvin + konsorten GmbH (info@konsorten.de)

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the 'Software'), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
