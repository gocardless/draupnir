# Add any extra build dependencies we need for running draupnir tests
FROM circleci/golang:1.14

RUN set -x \
    && sudo apt-get update \
    && sudo apt-get install -y \
        build-essential \
        ruby-dev \
    && sudo gem install bundler fpm \
    && sudo apt-get clean -y \
    && sudo rm -rf /var/lib/apt/lists/*
