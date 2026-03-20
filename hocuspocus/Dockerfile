FROM node:20-alpine

WORKDIR /app

COPY package.json ./
RUN npm install

COPY src/ ./src/
COPY tsconfig.json ./

RUN npx tsc

EXPOSE 1234

CMD ["node", "dist/index.js"]
