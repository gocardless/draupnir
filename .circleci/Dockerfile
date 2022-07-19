# Add any extra build dependencies we need for running draupnir tests
FROM cimg/go:1.17

RUN set -x \
    && sudo apt-get update \
    && sudo apt-get install -y \
        build-essential \
        ruby-dev \
    && sudo gem install bundler fpm \
    && sudo apt-get clean -y \
    && sudo rm -rf /var/lib/apt/lists/*
