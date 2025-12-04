# Stage 1: Build the React app
FROM node:16-alpine as builder

WORKDIR /app

COPY package.json ./
COPY package-lock.json ./
RUN npm install

COPY . .
RUN npm run build

# Stage 2: Serve the app with Nginx
FROM nginx:stable-alpine

COPY --from=builder /app/build /usr/share/nginx/html

# Copy a basic nginx config
# In a real app, you'd have a more complex config for routing to the backend API
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
