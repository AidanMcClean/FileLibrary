import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Route, Routes, Link } from 'react-router-dom';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faDownload } from '@fortawesome/free-solid-svg-icons';
import UploadForm from './components/UploadForm';
import axios from 'axios';
import './App.css';

function Home() {
  const [categories, setCategories] = useState([]);
  const [pdfs, setPdfs] = useState([]);

  useEffect(() => {
    const fetchCategories = async () => {
      try {
        const response = await axios.get('http://localhost:8080/api/categories');
        setCategories(response.data);
      } catch (error) {
        console.error('Error fetching categories:', error);
      }
    };

    const fetchPdfs = async () => {
      try {
        const response = await axios.get('http://localhost:8080/api/pdfs');
        setPdfs(response.data);
      } catch (error) {
        console.error('Error fetching PDFs:', error);
      }
    };

    fetchCategories();
    fetchPdfs();
  }, []);

  const getPdfsByCategory = (categoryId) => {
    return pdfs.filter(pdf => pdf.category_id === categoryId);
  };

  const handleDownload = (pdfId) => {
    const url = `http://localhost:8080/api/pdfs/${pdfId}`;
    window.location.href = url;
  };

  return (
    <div className="content">
      <div className="categories-list">
        {categories.map(category => {
          const pdfsInCategory = getPdfsByCategory(category.id);
          if (pdfsInCategory.length === 0) {
            return null;
          }
          return (
            <div key={category.id} className="category">
              <h2 className="category-title">{category.name}</h2>
              <ul className="pdf-list">
                {pdfsInCategory.map(pdf => (
                  <li key={pdf.id} className="pdf-item">
                    {pdf.name}
                    <FontAwesomeIcon
                      icon={faDownload}
                      className="download-icon"
                      onClick={() => handleDownload(pdf.id)}
                    />
                  </li>
                ))}
              </ul>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function App() {
  return (
    <Router>
      <div className="App">
        <header className="header">
          <span></span>
          <Link to="/upload" className="link">Upload PDF</Link>
        </header>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/upload" element={<UploadForm />} />
        </Routes>
      </div>
    </Router>
  );
}

export default App;
