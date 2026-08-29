FROM golang:1.20.4-buster

ENV USER=docker
ENV UID=1000
ENV GID=1000

RUN mkdir -p /home/$USER/app

RUN addgroup --gid 1000 $USER

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/home/$USER" \
    --gid "$GID" \
    --no-create-home \
    --uid "$UID" \
    "$USER"

RUN apt-get update && apt-get install -y curl openssl tzdata && apt-get clean

COPY ./ "/home/$USER/app/"
RUN rm /home/$USER/app/.env

RUN chown -R $USER:$USER /home/$USER

ENV HOME="/home/$USER"

WORKDIR /home/$USER/app
USER $USER

EXPOSE 3001

CMD ["./build/server_linux"]
