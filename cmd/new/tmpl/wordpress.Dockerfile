FROM wordpress:php8.3-apache

RUN apt update && apt install -y less
RUN curl -O https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar
RUN chmod +x wp-cli.phar
RUN mv wp-cli.phar /usr/local/bin/wp

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["apache2-foreground"]
