# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
# SPDX-License-Identifier: Apache-2.0

FROM node:20-alpine as build

LABEL repository="https://github.com/cloudoperators/greenhouse/tree/main/ui/docker/Dockerfile" 

RUN apk add --no-cache git jq \
  && apk upgrade --no-cache --no-progress \
  && apk del --no-cache --no-progress apk-tools alpine-keys


WORKDIR /greenhouse/apps 

RUN mkdir -p /tmp/greenhouse

# copy all greenhouse apps to /tmp/greenhouse
ADD . /tmp/greenhouse/

# copy all greenhouse extensions to /tmp/greenhouse
RUN \
  cd /tmp && \
  git clone https://github.com/cloudoperators/greenhouse-extensions.git && \
  cp -r greenhouse-extensions/doop/ui /tmp/greenhouse/doop && \
  cp -r greenhouse-extensions/alerts/ui /tmp/greenhouse/alerts && \
  cp -r greenhouse-extensions/heureka/ui /tmp/greenhouse/heureka && \
  rm -rf greenhouse-extensions


# run for each folder npm run build --if-present
RUN cd /tmp/greenhouse && for d in */; do (cd "$d" && npm install && npm run build --if-present); done

# run for each copy build to /dist

RUN cd /tmp/greenhouse && \
  for d in */; do \
  if [ -d "$d/build" ] && [ -n "$d/package.json" ] ; then \
  name=$(jq -r '.name' "$d/package.json"); \
  version=$(jq -r '.version' "$d/package.json"); \
  echo "Copying $name"; \
  mkdir -p "/greenhouse/apps/$name@$version"; \
  cp -rf "$d/build" "/greenhouse/apps/$name@$version/"; \
  cp -f "$d/package.json" "/greenhouse/apps/$name@$version/" ; \
  fi; \
  done

FROM nginx:alpine

RUN apk add --no-cache jq \
  && apk upgrade --no-cache --no-progress \
  && apk del --no-cache --no-progress apk-tools alpine-keys

WORKDIR /usr/share/nginx/html

ENV CURRENT_HOST=""
ENV OIDC_ISSUER_URL=""
ENV OIDC_CLIENT_ID=""
ENV K8S_API_ENDPOINT=""
ENV ENVIRONMENT=""

# default appProps.json
RUN cat <<EOF > /appProps.json
{
  "currentHost": "origin",
  "apiEndpoint": "https://api.endpoint.com",
  "environment": "dev",
  "authIssuerUrl": "https://auth.endpoint.com",
  "authClientId": "clientID"
}
EOF


COPY docker/index.html .
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf

COPY --from=build /greenhouse/apps /usr/share/nginx/html/apps
COPY --from=build /tmp/greenhouse/dashboard/public/favicon.ico /usr/share/nginx/html/favicon.ico

# Create a manifest file
ADD docker/generate_manifest.sh /usr/local/bin/generate_manifest.sh 
RUN mkdir -p extensions && chmod +x /usr/local/bin/generate_manifest.sh

ADD docker/entrypoint.sh /usr/local/bin/entrypoint.sh 
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT [ "/usr/local/bin/entrypoint.sh" ]

# Define the default command to run (could be overridden)
CMD ["nginx", "-g", "daemon off;"]
