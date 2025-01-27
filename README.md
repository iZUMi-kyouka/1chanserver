# README

1chan is a minimal forum app built by NextJS. Some other technologies used in building this app include Redux and Material UI.

You need to have **Docker, Docker Compose and Git** installed in your system to follow through with the local deployment.

If you are running Windows, deploying locally in Ubuntu using WSL is recommended.

On Debian-based Linux distributions, run the following commands to get all of them installed.

```jsx
sudo apt install git
sudo apt install docker
sudo apt install docker-compose
```

## Local Deployment

1. Clone the backend source code from https://github.com/iZUMi-kyouka/1chanserver into a directory of your choosing (hereafter, this directory will be referred to as “A”). Ensure you are at directory A, then run the following command (note the dot at the end).

```jsx
git clone https://github.com/iZUMi-kyouka/1chanserver.git .
```

2. Create a directory named “frontend” inside “A”.
3. Ensure you are at A/frontend. Clone the frontend source code from https://github.com/iZUMi-kyouka/1chanclient.git into the directory “frontend” by running the following command (note the dot at the end)

```jsx
git clone https://github.com/iZUMi-kyouka/1chanclient.git .
```

4. Go back to directory A, and run the following command:

```jsx
docker-compose build
```

5. After the containers have been successfully built, run the following command to start the container

```jsx
docker-compose up
```

If you encounter an error shown in Docker’s log as something along the line of the server unable to connect to the database, restart the container by pressing Ctrl+C, and then running the last command in step 5.