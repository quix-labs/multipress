# Multipress Setup Guide

Multipress is a powerful tool to manage and deploy WordPress instances with ease. Follow these steps to get started:

---

## **1. Download Multipress**

Download the latest release of Multipress from GitHub:

```bash
wget https://github.com/quix-labs/multipress/releases/latest/download/multipress
```

---

## **2. Make Multipress Executable**

Make the downloaded file executable:

```bash 
chmod u+x ./multipress
```

---

## **3. Add Multipress to Your System Path**

Move Multipress to a directory in your system's `PATH` (e.g., `/usr/local/bin`):

```bash 
sudo mv ./multipress /usr/local/bin
```

Confirm it’s accessible by running:

```bash 
multipress --help
```

---

## **4. Install Requirements**

Check all requirement and auto install them

```bash 
multipress doctor
```

---

## **5. Create a New Project**

Run the following command and follow the prompts to set up your project:

```bash 
multipress new
cd multipress
```

---

> All the subcommand must be run into your project directory

## **6. Deploy the Project**

Navigate to your project directory and deploy:

```bash 
multipress deploy
```

---

## **7. Configure Your WordPress Model**

After deployment, configure your WordPress instance as desired.

---

## **8. Generate Multiple Instances**

Replicate your WordPress instance as needed:

```bash 
multipress replicate 10
```

> Replace `10` with the number of instances you want to generate.

---

## **9. Generate All backups**

Generate backup of all your instances:

```bash 
multipress backup
```

---
You're all set! 🎉


# Additional commands

* Stop project: `multipress down`
* Start project: `multipress up`

# Removing project
1. Go to your project directory: `cd your_project`
2. Stop all containers: `multipress down`
3. You can now remove all the project folder (cannot be recovered): `rm -r ./your_project`
