import React, { useEffect, useState } from "react";
import "../styles/User.css"; // Importar el archivo CSS

const User = () => {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [list, setUserList] = useState([]);
  const [editingUser, setEditingUser] = useState(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    const response = await fetch(`${import.meta.env.VITE_API}/users`, {
      method: "POST",
      body: JSON.stringify({ name, email }),
      headers: {
        "Content-Type": "application/json",
      },
    });
    userList();
  };

  const updateSubmit = async (e) => {
    e.preventDefault();
    const response = await fetch(
      `${import.meta.env.VITE_API}/users/${editingUser.Id}`,
      {
        method: "PUT",
        body: JSON.stringify({ name, email }),
        headers: {
          "Content-Type": "application/json",
        },
      }
    );

    if (response.ok) {
      console.log("User updated successfully");
      // Realizar alguna acción adicional después de la actualización
      closeModal();
      userList();
    } else {
      console.error("Failed to update user");
      // Manejar el caso de error durante la actualización
    }
  };

  const deleteUser = async (e) => {
    e.preventDefault();
    console.log(editingUser);
    const response = await fetch(
      `${import.meta.env.VITE_API}/users/${editingUser.Id}`,
      {
        method: "DELETE",
        body: JSON.stringify({ name, email }),
        headers: {
          "Content-Type": "application/json",
        },
      }
    );

    if (response.ok) {
      console.log("User updated successfully");
      // Realizar alguna acción adicional después de la actualización
      closeModal();
      userList();
    } else {
      console.error("Failed to update user");
      // Manejar el caso de error durante la actualización
    }
  };

  const openModal = async (user) => {
    setEditingUser(user);
    setIsModalOpen(true);
    setName(user.name);
    setEmail(user.email);
  };

  const closeModal = () => {
    setEditingUser(null);
    setIsModalOpen(false);
    setName("");
    setEmail("");
  };

  const userList = async () => {
    const response = await fetch(`${import.meta.env.VITE_API}/users`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
      },
    });

    const data = await response.json();
    setUserList(data.data);
  };

  useEffect(() => {
    userList();
  }, []);

  return (
    <>
    <div className="container">
      <h1>Agregar nuevo usuario</h1>
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label htmlFor="name">Name:</label>
          <input
            type="text"
            id="name"
            name="name"
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <div className="form-group">
          <label htmlFor="email">Email:</label>
          <input
            type="email"
            id="email"
            name="email"
            placeholder="js@gmail.com"
            onChange={(e) => setEmail(e.target.value)}
          />
        </div>
        <div className="form-group">
          <button type="submit" className="btn-primary">
            Guardar
          </button>
        </div>
      </form>
    </div>
    <div className="container">
      <h1>Lista de maradonas</h1>
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Email</th>
            <th>Edit</th>
          </tr>
        </thead>
        <tbody>
          {list.map((item, index) => (
            <tr key={index}>
              <td>{item.name}</td>
              <td>{item.email}</td>
              <td>
                <button onClick={() => openModal(item)}>Editar</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
    {isModalOpen && editingUser && (
      <div className="modal">
        <div className="modal-content">
          <h1>Edit your data</h1>
          <form onSubmit={updateSubmit}>
            <div className="form-group">
              <label htmlFor="name">Name:</label>
              <input
                type="text"
                id="name"
                className="form-control"
                value={name || ""}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label htmlFor="email">Email:</label>
              <input
                type="email"
                id="email"
                className="form-control"
                value={email || ""}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div className="button-group">
              <button type="submit" className="btn-primary">
                Submit
              </button>
              <div className="button-group-right">
                <button className="btn-cancel" onClick={closeModal}>
                  Cancel
                </button>
                <button className="btn-delete" onClick={deleteUser}>
                  Delete
                </button>
              </div>
            </div>
          </form>
        </div>
      </div>
    )}
  </>  
  );
};

export default User;
