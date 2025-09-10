cd $(dirname $(readlink -f "$0"))
cd ../src-go
CGO_ENABLED=0 go build -tags=with_gvisor -trimpath -ldflags "-X github.com/legiz-ru/prizrak-box/api.Version=v-test" -o px