# Update the NODE_VERSION arg in docker-compose.yml to pick a Node version: 10, 12, 14
ARG NODE_VERSION=14
FROM mcr.microsoft.com/vscode/devcontainers/javascript-node:0-${NODE_VERSION}

# VARIANT can be either 'hugo' for the standard version or 'hugo_extended' for the extended version.
ARG VARIANT=hugo
# VERSION can be either 'latest' or a specific version number
ARG VERSION=latest

# Download Hugo
RUN apt-get update && apt-get install -y ca-certificates openssl git curl && \
    rm -rf /var/lib/apt/lists/* && \
    case ${VERSION} in \
    latest) \
    export VERSION=$(curl -s https://api.github.com/repos/gohugoio/hugo/releases/latest | grep "tag_name" | awk '{print substr($2, 3, length($2)-4)}') ;;\
    esac && \
    echo ${VERSION} && \
    wget -O ${VERSION}.tar.gz https://github.com/gohugoio/hugo/releases/download/v${VERSION}/${VARIANT}_${VERSION}_Linux-64bit.tar.gz && \
    tar xf ${VERSION}.tar.gz && \
    mv hugo /usr/bin/hugo

# blogposter port
EXPOSE 8080

# [Optional] Uncomment this section to install additional OS packages you may want.
#
RUN wget https://github.com/jgm/pandoc/releases/download/2.17.0.1/pandoc-2.17.0.1-1-amd64.deb && sudo dpkg -i ./pandoc-2.17.0.1-1-amd64.deb
USER node
RUN git clone https://github.com/sarahlehman/smalltownkitten.git /home/node/smalltownkitten-dev
RUN cd /home/node/smalltownkitten-dev && npm install

# [Optional] Uncomment if you want to install more global node packages
# RUN sudo -u node npm install -g <your-package-list-here>
