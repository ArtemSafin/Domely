FROM nginx:alpine

COPY services/frontend-service/nginx.conf /etc/nginx/conf.d/default.conf
COPY services/frontend-service/index.html /usr/share/nginx/html/index.html
COPY services/frontend-service/manifest.json /usr/share/nginx/html/manifest.json

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]