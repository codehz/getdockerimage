package getdockerimage

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

type authResp struct {
	Token string `json:"token"`
}

type manifest struct {
	FsLayers []struct {
		BlobSum string `json:"blobSum"`
	} `json:"fsLayers"`
}

// GetDockerImage Request to get image
type GetDockerImage struct {
	Registry    string
	Repo        string
	Token       string
	ReaderProxy func(io.Reader) io.Reader
}

// Auth to docker
func (ctx *GetDockerImage) Auth() error {
	resp, err := http.Get(fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull", ctx.Repo))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	auth := authResp{}
	if err := json.Unmarshal(contents, &auth); err != nil {
		return err
	}
	ctx.Token = auth.Token
	return nil
}

func fetch(url string, token string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return contents, nil
}

// FetchLayers fetch docker layers
func (ctx GetDockerImage) FetchLayers() ([]string, error) {
	contents, err := fetch(fmt.Sprintf("%s/%s/manifests/latest", ctx.Registry, ctx.Repo), ctx.Token)
	if err != nil {
		return nil, err
	}
	info := manifest{}
	if err := json.Unmarshal(contents, &info); err != nil {
		return nil, err
	}
	layers := make([]string, len(info.FsLayers))
	for i, layer := range info.FsLayers {
		layers[i] = layer.BlobSum
	}
	return layers, nil
}

// DownloadLayer download layer
func (ctx GetDockerImage) DownloadLayer(blob string, target string) {
	out, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	defer out.Close()
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s/blobs/%s", ctx.Registry, ctx.Repo, blob), nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", "Bearer "+ctx.Token)
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, ctx.ReaderProxy(resp.Body))
	if err != nil {
		panic(err)
	}
}
