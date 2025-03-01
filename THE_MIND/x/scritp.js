const statusDiv = document.getElementById('status');
const levelDiv = document.getElementById('level');
const livesDiv = document.getElementById('lives');
const shurikensDiv = document.getElementById('shurikens');
const playerCardsDiv = document.getElementById('player-cards');
const tableCardsDiv = document.getElementById('table-cards');
const concentrationButton = document.getElementById('concentration-button');
const shurikenButton = document.getElementById('shuriken-button');
const readyButton = document.getElementById('ready-button');
const rulesContentDiv = document.getElementById('rules-content');
const playersTableBody = document.querySelector('#players-table tbody');
const ws = new WebSocket('ws://192.168.10.130:8085/ws');
const spinShuriken = document.getElementById('spin-shuriken');
let playerCards = [];
let players = {};
let playerId = null;
let isRotating = false;

ws.onopen = () => {
    statusDiv.textContent = 'Connecté au serveur';
    console.log('Connecté au serveur');
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log('Message reçu :', data);

    if (data.type === 'init') {
        playerCards = data.cards;
        levelDiv.textContent = `Niveau: ${data.level}`;
        livesDiv.textContent = `Vies restantes: ${data.lives}`;
        shurikensDiv.textContent = `Shurikens restants: ${data.shurikens}`;
        playerCardsDiv.innerHTML = '';
        playerCards.forEach(card => {
            const cardDiv = document.createElement('div');
            cardDiv.className = 'card';
            cardDiv.textContent = card.Number;
            cardDiv.style.backgroundColor = card.Color;
            cardDiv.onclick = () => playCard(card);
            playerCardsDiv.appendChild(cardDiv);
        });
        statusDiv.textContent = `Vous avez ${playerCards.length} cartes`;
        playerId = data.playerId;
        updatePlayersTable(data.players);
    } else if (data.type === 'play') {
        const cardDiv = document.createElement('div');
        cardDiv.className = 'card';
        cardDiv.textContent = data.card;
        cardDiv.style.backgroundColor = data.color;
        tableCardsDiv.appendChild(cardDiv);
    } else if (data.type === 'status') {
        statusDiv.textContent = data.message;
    } else if (data.type === 'clear') {
        tableCardsDiv.innerHTML = '';
    } else if (data.type === 'concentration_status') {
        updateConcentrationButton(data.status);
    } else if (data.type === 'update') {
        playerCards = data.cards;
        playerCardsDiv.innerHTML = '';
        playerCards.forEach(card => {
            const cardDiv = document.createElement('div');
            cardDiv.className = 'card';
            cardDiv.textContent = card.Number;
            cardDiv.style.backgroundColor = card.Color;
            cardDiv.onclick = () => playCard(card);
            playerCardsDiv.appendChild(cardDiv);
        });
    } else if (data.type === 'players_update') {
        updatePlayersTable(data.players);
    }
};

ws.onclose = () => {
    statusDiv.textContent = 'Déconnecté du serveur';
    console.log('Déconnecté du serveur');
};

ws.onerror = (error) => {
    statusDiv.textContent = 'Erreur WebSocket : ' + error.message;
    console.log('Erreur WebSocket :', error.message);
};

function playCard(card) {
    console.log('Playing card:', card);
    ws.send(JSON.stringify({ type: 'play', card: card.Number, color: card.Color }));
    playerCards = playerCards.filter(c => c.Number !== card.Number);
    playerCardsDiv.innerHTML = '';
    playerCards.forEach(c => {
        const cardDiv = document.createElement('div');
        cardDiv.className = 'card';
        cardDiv.textContent = c.Number;
        cardDiv.style.backgroundColor = c.Color;
        cardDiv.onclick = () => playCard(c);
        playerCardsDiv.appendChild(cardDiv);
    });
}

concentrationButton.addEventListener('click', () => {
    const currentStatus = concentrationButton.dataset.status;
    const newStatus = currentStatus === 'concentration' ? 'game' : 'concentration';
    console.log('Toggling concentration:', newStatus);
    ws.send(JSON.stringify({ type: 'concentration', concentration: newStatus === 'game' }));
    updateConcentrationButton(newStatus);
});

shurikenButton.addEventListener('click', () => {
    console.log('Using shuriken');
    ws.send(JSON.stringify({ type: 'shuriken' }));
});

readyButton.addEventListener('click', () => {
    console.log('Ready button clicked');
    ws.send(JSON.stringify({ type: 'ready' }));
});

function updateConcentrationButton(status) {
    concentrationButton.dataset.status = status;
    if (status === 'game') {
        concentrationButton.textContent = 'ACTION';
        concentrationButton.style.background = 'linear-gradient(145deg, #008000, #006440)';
    } else {
        concentrationButton.textContent = 'On se concentre';
        concentrationButton.style.background = 'linear-gradient(145deg, #ff4500, #ffa500)';
    }
    concentrationButton.classList.remove('clicked');
}

function updatePlayersTable(players) {
    playersTableBody.innerHTML = '';
    for (let i = 0; i < 10; i++) {
        const row = document.createElement('tr');
        const connectedCell = document.createElement('td');
        const readyCell = document.createElement('td');
        const playerNumberCell = document.createElement('td');

        playerNumberCell.textContent = i + 1;

        if (players[i]) {
            if (players[i].ready) {
                const readyDot = document.createElement('div');
                readyDot.className = 'ready';
                readyCell.appendChild(readyDot);
            } else {
                const connectedDot = document.createElement('div');
                connectedDot.className = 'connected';
                connectedCell.appendChild(connectedDot);
            }
        }

        row.appendChild(connectedCell);
        row.appendChild(readyCell);
        row.appendChild(playerNumberCell);
        playersTableBody.appendChild(row);
    }
}

// Fetch and display the rules from the Markdown file
fetch('/THE_MIND/rules.md')
    .then(response => response.text())
    .then(text => {
        rulesContentDiv.innerHTML = marked.parse(text);
    })
    .catch(error => {
        rulesContentDiv.textContent = 'Impossible de charger les règles.';
        console.error('Erreur de chargement des règles:', error);
    });

// Add event listeners for spin interaction
spinShuriken.addEventListener('mousedown', startRotation);
spinShuriken.addEventListener('touchstart', startRotation);
spinShuriken.addEventListener('mouseup', stopRotation);
spinShuriken.addEventListener('touchend', stopRotation);

function startRotation(event) {
    event.preventDefault();
    isRotating = true;
    spinShuriken.classList.add('rotating');
}

function stopRotation(event) {
    event.preventDefault();
    isRotating = false;
    spinShuriken.classList.remove('rotating');
}

