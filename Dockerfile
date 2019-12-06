FROM library/golang AS build

MAINTAINER tailinzhang1993@gmail.com

ENV GO111MODULE off
ENV APP_DIR /go/src/fabric-byzantine
RUN mkdir -p $APP_DIR
WORKDIR $APP_DIR
ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o fabric-byzantine .
ENTRYPOINT ./fabric-byzantine

# Create a minimized Docker mirror
FROM scratch AS prod

COPY --from=build /go/src/fabric-byzantine/fabric-byzantine /fabric-byzantine
EXPOSE 8080
CMD ["/fabric-byzantine"]
