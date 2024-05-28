import React, { useState, useEffect } from 'react';
import axios from 'axios';
import './RemovePDF.css';

function RemovePDF() {
  const [categories, setCategories] = useState([]);
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [pdfs, setPdfs] = useState([]);
  const [selectedPdfs, setSelectedPdfs] = useState([]);
  const [apiKey, setApiKey] = useState('');

  useEffect(() => {
    fetchCategories();
    fetchPdfs();
  }, []);

  const fetchCategories = async () => {
    try {
      const response = await axios.get('https://aidanlibrarymanagementapp.azurewebsites.net/api/categories');
      setCategories(response.data);
    } catch (error) {
      console.error('Error fetching categories:', error);
    }
  };

  const fetchPdfs = async () => {
    try {
      const response = await axios.get('https://aidanlibrarymanagementapp.azurewebsites.net/api/pdfs');
      setPdfs(response.data);
    } catch (error) {
      console.error('Error fetching PDFs:', error);
    }
  };

  const handleCategoryChange = (event) => {
    setSelectedCategory(event.target.value);
  };

  const handleCheckboxChange = (pdfId) => {
    setSelectedPdfs((prevSelectedPdfs) =>
      prevSelectedPdfs.includes(pdfId)
        ? prevSelectedPdfs.filter((id) => id !== pdfId)
        : [...prevSelectedPdfs, pdfId]
    );
  };

  const handleRemovePdfs = async () => {
    try {
      await axios.post('https://aidanlibrarymanagementapp.azurewebsites.net/api/remove-pdfs', {
        pdfIds: selectedPdfs,
      }, {
        headers: {
          'x-api-key': apiKey
        }
      });
      fetchPdfs();
      setSelectedPdfs([]);
    } catch (error) {
      console.error('Error removing PDFs:', error);
    }
  };

  const filteredPdfs = selectedCategory === 'all'
    ? pdfs
    : pdfs.filter((pdf) => pdf.category_id === parseInt(selectedCategory));

  return (
    <div className="remove-pdf-container">
      <h2>Remove PDFs</h2>
      <div className="filter-container">
        <label htmlFor="category">Category:</label>
        <select
          id="category"
          value={selectedCategory}
          onChange={handleCategoryChange}
        >
          <option value="all">All Categories</option>
          {categories.map((category) => (
            <option key={category.id} value={category.id}>
              {category.name}
            </option>
          ))}
        </select>
      </div>
      <div className="api-key-container">
        <label htmlFor="api-key">API Key:</label>
        <input
          type="password"
          id="api-key"
          value={apiKey}
          onChange={(e) => setApiKey(e.target.value)}
          placeholder="Enter API Key"
          className="input-field"
        />
      </div>
      <ul className="pdf-list">
        {filteredPdfs && filteredPdfs.map((pdf) => (
          <li key={pdf.id} className="pdf-item">
            <input
              type="checkbox"
              checked={selectedPdfs.includes(pdf.id)}
              onChange={() => handleCheckboxChange(pdf.id)}
            />
            {pdf.name}
          </li>
        ))}
      </ul>
      <button onClick={handleRemovePdfs} className="remove-button">
        Remove Selected PDFs
      </button>
    </div>
  );
}

export default RemovePDF;
