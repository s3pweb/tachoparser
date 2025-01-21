# Nom de l'image et tag
IMAGE_NAME = s3pweb/dddhttp
IMAGE_TAG = 0.0.1

# Cible pour construire l'image Docker
build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

# Cible pour exécuter le conteneur Docker
run:
	docker run -p 8080:8080 $(IMAGE_NAME):$(IMAGE_TAG)

# Cible pour supprimer l'image Docker
clean:
	docker rmi $(IMAGE_NAME):$(IMAGE_TAG)

# Cible par défaut
default: build