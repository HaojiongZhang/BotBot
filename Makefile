# Default target
.PHONY: run add commit ssh-add push

# Run the Go application
run:
	go run cmd/main.go

# Add changes to staging area
add:
	git add .

# Commit changes with a message from user input
commit:
	@read -p "Enter commit message: " msg; \
	git commit -m "$$msg"

# Add SSH key
ssh-add:
	ssh-add ~/.ssh/id_rsa

# Push changes to origin main
push: add commit ssh-add
	git push origin main