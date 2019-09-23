module github.com/erik13121/ssh-cert-authority

require (
	cloud.google.com/go v0.46.3
	github.com/aws/aws-sdk-go v1.15.76
	github.com/codegangsta/cli v1.20.0
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/handlers v1.4.0
	github.com/gorilla/mux v1.6.2
	github.com/stretchr/testify v1.4.0 // indirect
	go.opencensus.io v0.22.1 // indirect
	golang.org/x/crypto v0.0.0-20190605123033-f99c8df09eb5
	google.golang.org/genproto v0.0.0-20190911173649-1774047e7e51
)

replace github.com/erik13121/ssh-cert-authority v0.0.0-20190922150606-1266cbce11d0 => github.com/erik13121/ssh-cert-authority v0.0.0-20190831142453-68269dffd75a

go 1.13
