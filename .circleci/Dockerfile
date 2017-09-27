FROM circleci/golang:1.9

RUN set -x \
	  && sudo apt-get update \
		&& sudo apt-get install -y \
		    build-essential \
				ruby-dev \
		&& sudo gem install bundler \
		&& sudo apt-get clean -y \
		&& sudo rm -rf /var/lib/apt/lists/*
