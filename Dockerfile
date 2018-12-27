FROM opeo/go-alpine

WORKDIR /go/src/go-panda

COPY . .

# Use pre-compiled git and mercurial from base image
RUN go get -d \
    && apk del git mercurial

# Run at 10am everyday
RUN chmod -R 777 /go/src/go-panda && \
echo "0  10  *  *  *  /go/src/go-panda/cronjob" > /etc/crontabs/root 

CMD crond -l 2 -f