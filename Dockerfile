FROM ndm:deps

RUN mkdir -p /usr/local/src/
WORKDIR /usr/local/src/
RUN git clone https://github.com/neurodata/DataManager.git
WORKDIR /usr/local/src/DataManager
RUN git checkout alex/mesh
RUN mkdir -p build

WORKDIR /usr/local/src/DataManager/build
# Note that tests do not yet build statically
RUN LD_LIBRARY_PATH=/usr/local/src/folly/follylib/lib:$LD_LIBRARY_PATH cmake \
   -DCMAKE_BUILD_TYPE=release \
   -DENABLE_TESTS=off \
   -DGTEST_ROOT=/usr/src/gtest \ 
   -DUSE_STATIC_LIBS=on ..  
RUN make -j $(nproc)
RUN make install

# Install go
RUN mkdir -p /usr/local/src/golang
WORKDIR /usr/local/src/golang
RUN wget https://storage.googleapis.com/golang/go1.9.2.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.9.2.linux-amd64.tar.gz
ENV PATH="${PATH}:/usr/local/go/bin"

# Setup Go Path
ENV GOPATH="/usr/local/src/go"

# Copy in go files
RUN mkdir -p /usr/local/src/go/src
COPY . /usr/local/src/go/src/github.com/neurodata/ndmesh
WORKDIR /usr/local/src/go/src/github.com/neurodata/ndmesh
RUN go build
RUN go install

ENV PATH="${PATH}:/usr/local/src/go/bin"
ENV LD_LIBRARY_PATH="${LD_LIBRARY_PATH}:/usr/local/lib"
WORKDIR /

FROM ubuntu:16.04

RUN apt-get update
RUN apt-get upgrade -y 

# Even compiled statically, NDM still needs a few libraries
RUN apt-get install -y \ 
    libjbig-dev \
    libjpeg-dev 

# Go dependencies (go prefers dynamic linking)
RUN apt-get install -y \ 
    libboost-all-dev \
    libgoogle-glog-dev \
    ca-certificates

COPY --from=0 /usr/local/include /usr/local/include
COPY --from=0 /usr/local/lib /usr/local/lib
COPY --from=0 /usr/local/src/go/bin /usr/local/bin

ENV PATH="${PATH}:/usr/local/bin"
ENV LD_LIBRARY_PATH="${LD_LIBRARY_PATH}:/usr/local/lib"
WORKDIR /

CMD ["/usr/local/bin/ndmesh"]
