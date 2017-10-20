FROM golang:1.9
WORKDIR /opt/gendry
ADD ./gendry.tar.gz ./
EXPOSE 80
ENTRYPOINT ["./bin/gendry"]
CMD ["-help"]
