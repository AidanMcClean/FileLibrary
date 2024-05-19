import React from 'react';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
import UploadForm from './components/UploadForm';
import CategoryList from './components/CategoryList';
import './App.css';

function Home() {
  return (
    <div className="content">
      <h1>Welcome to the Library</h1>
      <CategoryList />
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
