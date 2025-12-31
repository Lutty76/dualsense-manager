# DualSense Controller Manager (Linux)

Une application performante en **Go** avec une interface graphique **Fyne** pour gérer et surveiller vos manettes Sony DualSense sur Linux.

## 🚀 Fonctionnalités

- **Multi-Manettes :** Gestion indépendante via un système d'onglets (réglages commun).
- **Surveillance de la Batterie :**
    - Affichage du pourcentage et de l'état (Charge/Décharge) en temps réel.
    - Animation de progression sur les LEDs Player pendant la charge.
    - Effet de "respiration" (Breathe) sur la barre lumineuse RGB pendant la charge.
- **Personnalisation des LEDs :**
    - **Mode Batterie :** Les LEDs Player et RGB reflètent le niveau de charge (Vert -> Orange -> Rouge).
    - **Mode Joueur :** Affiche le numéro physique de la manette (1, 2, 3, 4).
    - **Mode Fixe :** Bleu fixe (selecteur a venir)
- **Gestion de l'inactivité :**
    - Détection d'activité précise sur les boutons et les axes.
    - Gestion de la **Deadzone** paramétrable pour éviter le "stick drift".
    - Compte à rebours avant mise en veille.

## 🛠 Installation

### 1. Dépendances
Assurez-vous d'avoir les bibliothèques de développement installées (pour Fyne/OpenGL) :

```bash
# Sur Fedora
sudo dnf install golang libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel mesa-libGL-devel libXi-devel libXxf86vm-devel

# Sur Ubuntu/Debian
sudo apt install golang libgl1-mesa-dev xorg-dev
```
### 2. Configuration des permissions (UDEV)

Créez le fichier suivant pour autoriser l'accès aux LEDs et au Joystick sans droits root :

`sudo nano /etc/udev/rules.d/999-dualsense.rules`

Contenu du fichier :
Extrait de code

```bash
# Règle pour les LEDs Player (Blanches)
SUBSYSTEM=="leds", KERNEL=="*player*", MODE="0666", RUN+="/bin/chmod 666 %S%p/brightness %S%p/trigger"

# Règle pour l'Indicateur (RGB) - On ajoute multi_intensity
SUBSYSTEM=="leds", KERNEL=="*indicator*", MODE="0666", RUN+="/bin/chmod 666 %S%p/brightness %S%p/trigger %S%p/multi_intensity"

```







Appliquez les changements :

`sudo udevadm control --reload-rules && sudo udevadm trigger`

### 🏗 Architecture

Le projet est structuré pour éviter les dépendances cycliques :

    - /internal/ui : Interface Fyne et gestion des onglets. Utilise des callbacks pour notifier les changements au service hardware.

    - /internal/service : Moteur hardware (lecture /dev/input/jsX et écriture dans /sys/class/leds).

    - /internal/config : Gestion de la persistance YAML des réglages utilisateur.

### 🖥 Utilisation

    Lancez l'application : go run main.go

    Les manettes sont détectées automatiquement et apparaissent dans des onglets séparés.

    Ajustez la Deadzone si le compteur d'inactivité ne se déclenche pas à cause d'un stick usé.

⚖ Licence

MIT