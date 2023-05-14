FROM golang:1.20 as base

FROM base as built

WORKDIR /opt/app/api
COPY . .

RUN go get -d -v ./...
RUN go build -o /usr/bin/api-server ./*.go

CMD ["api-server"]