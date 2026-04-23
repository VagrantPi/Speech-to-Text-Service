FROM nginx:alpine

COPY apps/web/index.html /usr/share/nginx/html/index.html

COPY apps/web/nginx.conf /etc/nginx/nginx.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]