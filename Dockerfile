FROM node:13

EXPOSE 2000
ENV REQUIRE_HTTPS=false

USER node
RUN mkdir -p /home/node/codenames
WORKDIR /home/node/codenames

COPY --chown=node * ./
COPY --chown=node server/ ./server/
COPY --chown=node public/ ./public/

RUN npm install

CMD npm start
