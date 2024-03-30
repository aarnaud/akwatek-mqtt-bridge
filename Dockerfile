FROM golang:alpine AS builderimage
WORKDIR /go/src/akwatek-mqtt-bridge
COPY . .
RUN go build -o akwatek-mqtt-bridge main.go


###################################################################

FROM alpine
COPY --from=builderimage /go/src/akwatek-mqtt-bridge/akwatek-mqtt-bridge /app/
WORKDIR /app
CMD ["./akwatek-mqtt-bridge"]
