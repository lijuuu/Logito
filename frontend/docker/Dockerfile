# Simple React frontend build
FROM node:20-alpine

WORKDIR /app

# Copy package files
COPY package.json package-lock.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Install a simple HTTP server
RUN npm install -g serve

# Expose port 3030
EXPOSE 3030

# Serve the built application
CMD ["serve", "-s", "build", "-l", "3030"]