# Build the image with go-1.11.1
FROM golang:1.11.1 AS build

# Copy all the files needed for the build to the container
RUN mkdir -p /go/src/github.com/su225/raft/

# Build the binary of the raft node with CGO disabled. This
# will produce a single binary with all libraries linked
WORKDIR /go/src/github.com/su225/raft

# Enable go modules
ENV GO111MODULE=on

# Copy the go.mod and go.sum files and then download
# the dependencies from them. This is because if both of
# those files do not change then the layer downloading
# and storing dependencies is also cached
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .

# Finally build the project to produce the binary which will
# then be moved to another container for the launch
RUN CGO_ENABLED=0 GOOS=linux \
    go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o raft

# Once build is successful copy the binary to another container
# and run it with appropriate settings
FROM alpine:3.9

# Create a new user to run the raft node and
# switch to that user
RUN mkdir node && \
    mkdir -p node/cluster-data/log/data && \
    mkdir -p node/cluster-data/log/metadata && \
    mkdir -p node/cluster-data/state && \
    mkdir -p node/cluster-data/snapshot && \
    mkdir -p node/cluster-data/cluster
    
# Define the volumes for this container
# that must be supplied from the host
VOLUME node/cluster-data/log
VOLUME node/cluster-data/state
VOLUME node/cluster-data/snapshot

# Copy the binary from the build container
COPY --from=build /go/src/github.com/su225/raft/raft /node
COPY --from=build /go/src/github.com/su225/raft/scripts/node_start.sh /node
WORKDIR /node

# Launch the process from the run-script
CMD sh ./node_start.sh

# Start the node with the settings specified in the container
# environment and expose appropriate ports for communication
EXPOSE 6666 7777