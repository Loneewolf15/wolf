<?php
  class Product {
    private $db;
    
    public function __construct(){
      $this->db = new Database;
    }

    // Get All Posts
    public function getPosts(){
      $this->db->query("SELECT *, 
                        posts.id as postId, 
                        initkey.id as userId
                        FROM posts 
                        INNER JOIN initkey 
                        ON posts.user_id = initkey.id
                        ORDER BY posts.created_at DESC;");

      $results = $this->db->resultset();

      return $results;
    }

    // Get Post By ID
    public function getPostById($id){
      $this->db->query("SELECT * FROM posts WHERE id = :id");

      $this->db->bind(':id', $id);
      
      $row = $this->db->single();

      return $row;
    }
    // Get Post By ID
     public function findcable()
{
    try {
        $this->db->query("SELECT * FROM CablePlans");
        $rows = $this->db->resultSet();

        // Check if any rows were returned
        if (!empty($rows)) {
            return $rows;
        } else {
            return null; // Or handle the case when no data is found
        }
    } catch (PDOException $e) {
        // Handle database query errors
        return "Error: " . $e->getMessage();
        // return null; // Or handle the error in an appropriate way
    }
}
     public function finddstv()
{
    try {
        $this->db->query("SELECT * FROM dstv_cable_plans");
        $rows = $this->db->resultSet();

        // Check if any rows were returned
        if (!empty($rows)) {
            return $rows;
        } else {
            return null; // Or handle the case when no data is found
        }
    } catch (PDOException $e) {
        // Handle database query errors
        return "Error: " . $e->getMessage();
        // return null; // Or handle the error in an appropriate way
    }
}
     public function findgotv()
{
    try {
        $this->db->query("SELECT * FROM gotv_cable_plans");
        $rows = $this->db->resultSet();

        // Check if any rows were returned
        if (!empty($rows)) {
            return $rows;
        } else {
            return null; // Or handle the case when no data is found
        }
    } catch (PDOException $e) {
        // Handle database query errors
        return "Error: " . $e->getMessage();
        // return null; // Or handle the error in an appropriate way
    }
}
     public function getAllProducts()
   {
    try {
        $this->db->query("SELECT * FROM products");
        $rows = $this->db->resultSet();

        // Check if any rows were returned
        if (!empty($rows)) {
            return $rows;
        } else {
            return null; // Or handle the case when no data is found
        }
    } catch (PDOException $e) {
        // Handle database query errors
        return "Error: " . $e->getMessage();
        // return null; // Or handle the error in an appropriate way
    }
  }
     public function getProducts($data1, $da)
   {
     try {
        $this->db->query("SELECT * FROM products WHERE seller_id = :seller_id AND shop_id = :shop_id");

        $this->db->bind(':seller_id', $data1);
        $this->db->bind(':shop_id', $da);

        $row = $this->db->resultSet();

        if (!empty($row)) {
            return $row;
        } else {
            return null; 
        }
        } catch (PDOException $e) {
            return "Error: " . $e->getMessage();

        }

    }
     public function getProducts2($data1, $da)
   {
     try {
        $this->db->query("SELECT * FROM product_wishlist WHERE seller_id = :seller_id AND shop_id = :shop_id");

        $this->db->bind(':seller_id', $data1);
        $this->db->bind(':shop_id', $da);

        $row = $this->db->resultSet();

        if (!empty($row)) {
            return $row;
        } else {
            return null; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }
     public function getProductById($da)
   {
     try {
        $this->db->query("SELECT * FROM products WHERE product_id = :product_id");

      
        $this->db->bind(':product_id', $da);

        $row = $this->db->single();

        if (!empty($row)) {
            return $row;
        } else {
            return null; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }
     public function getProductByUserId($da, $id)
   {
     try {
        $this->db->query("SELECT * FROM product_wishlist WHERE product_id = :product_id AND user_id = :user_id");

      
        $this->db->bind(':product_id', $da);
        $this->db->bind(':user_id', $id);

        $row = $this->db->resultSet();

        if (!empty($row)) {
            return $row;
        } else {
            return null; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }
    
     public function getWishList($id)
   {
     try {
        $this->db->query("SELECT * FROM product_wishlist WHERE user_id = :user_id");

        $this->db->bind(':user_id', $id);

        $row = $this->db->resultSet();

        if (!empty($row)) {
            return $row;
        } else {
            return null; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }
    
     public function getProductByUniCat($data1, $da)
   {
     try {
        $this->db->query("SELECT * FROM products WHERE uni = :uni AND product_cat = :product_cat");

        $this->db->bind(':uni', $data1);
        $this->db->bind(':product_cat', $da);

        $row = $this->db->resultSet();

        if (!empty($row)) {
            return $row;
        } else {
            return null; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }
    
 
    
    
     public function getProductById2($da)
   {
     try {
        $this->db->query("SELECT * FROM products WHERE product_id = :product_id");

        $this->db->bind(':product_id', $da);

        $row = $this->db->single();

        if (!empty($row)) {
            return $row;
        } else {
            return null; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }
     public function getUserDetails($da)
   {
     try {
        $this->db->query("SELECT * FROM initkey_rid WHERE user_id = :user_id");

        $this->db->bind(':user_id', $da);

        $row = $this->db->single();

        if (!empty($row)) {
            return $row;
        } else {
            return null; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }
     public function getProductById22($da, $de)
   {
     try {
        $this->db->query("SELECT * FROM product_wishlist WHERE product_id = :product_id AND user_id = :user_id");

        $this->db->bind(':product_id', $da);
        $this->db->bind(':user_id', $de);

        $row = $this->db->single();

        if (!empty($row)) {
            return $row;
        } else {
            return null; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }
     public function getProductReviews($da)
   {
     try {
        $this->db->query("SELECT * FROM reviewxx WHERE product_id = :product_id");

        $this->db->bind(':product_id', $da);

        $row = $this->db->resultSet();

        if (!empty($row)) {
            return $row;
        } else {
            return null; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }
   public function getAverageReview($da)
{
    try {
        // Prepare the query
        $this->db->query("SELECT SUM(rating) / COUNT(*) AS average FROM reviewxx WHERE product_id = :product_id");

        // Bind the product ID
        $this->db->bind(':product_id', $da);

        // Execute and fetch the result
        $row = $this->db->single(); // Use single() if you are expecting only one row

        // Check if a result was returned
        if (!empty($row)) {
            // Return the average value from the result
            return $row; // Return the 'average' column value
        } else {
            return null; 
        }
    } catch (PDOException $e) {
        return "Error: " . $e->getMessage();
    }
}

     public function getProductByUni($da)
   {
     try {
        $this->db->query("SELECT * FROM products WHERE uni = :uni");

        $this->db->bind(':uni', $da);

        $rows = $this->db->resultSet();

        if (!empty($rows)) {
            return $rows;
        } else {
            return null; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }
    
//      public function getProductByShopName($da)
//   {
//      try {
//         $this->db->query("SELECT * FROM products WHERE uni = :uni");

//         $this->db->bind(':uni', $da);

//         $rows = $this->db->resultSet();

//         if (!empty($rows)) {
//             return $rows;
//         } else {
//             return null; 
//         }
//         } catch (PDOException $e) {

//             return "Error: " . $e->getMessage();

//         }

//     }
    
    
public function getProductByShopName($da)
{
    try {
        $query = "SELECT * FROM shopdetails WHERE shop_tag = :shop_tag";
        $this->db->query($query);
        $this->db->bind(':shop_tag', $da);
        $row = $this->db->single();
// return $row;
        if (!empty($row)) {
            // Query products table
            $this->db->query("SELECT * FROM products WHERE seller_id = :seller_id");
            $this->db->bind(':seller_id', $row->seller_id);

            $rows = $this->db->resultSet();

            return !empty($rows) ? $rows : null;
        } else {
            return null; // No shop found
        }
    } catch (PDOException $e) {
        error_log("PDO Error: " . $e->getMessage());
        throw new RuntimeException('Database query failed');
    }
}

    


     public function findstartime()
 {
    try {
        $this->db->query("SELECT * FROM startime_cable_plans");
        $rows = $this->db->resultSet();

        // Check if any rows were returned
        if (!empty($rows)) {
            return $rows;
        } else {
            
            return null; // Or handle the case when no data is found
        }
    } catch (PDOException $e) {
        // Handle database query errors
        return "Error: " . $e->getMessage();
        // return null; // Or handle the error in an appropriate way
    }
  }
    public function getTransactions($id){
      $this->db->query("SELECT * FROM transactions WHERE userid = :userid ORDER BY transactions.date DESC;"  );

      $this->db->bind(':userid', $id);
      
      $resultset = $this->db->resultset();

      if ($resultset) {
        return $resultset;
      }else {
        return false;
      }
      
    }
    public function checkBalance($data) {
      // Prepare the SQL query
      $this->db->query("SELECT * FROM user_account WHERE user_id = :user_id");
  
      // Bind the user_id parameter
      $this->db->bind(':user_id', $data['user_id']);
    
      // Fetch the single row
      $row = $this->db->single();
  
      // Check if the account balance is sufficient
      if ($row && $row->acountbalance >= $data['amount']) {
          return true;
      } else {
          return false;
      }
  }
  
    public function getBal($data) {
      // Prepare the SQL query
      $this->db->query("SELECT * FROM user_account WHERE user_id = :user_id");
  
      // Bind the user_id parameter
      $this->db->bind(':user_id', $data['user_id']);
    
      // Fetch the single row
      $row = $this->db->single();
  
      // Check if the account balance is sufficient
      if ($row) {
          return $row->acountbalance;
      } else {
          return false;
      }
  }
  public function editProduct($data) {
    // Prepare Query
    $this->db->query('UPDATE products SET 
        product_name = :product_name, 
        product_desc = :product_desc, 
        product_cat = :product_cat, 
        product_img1 = :product_img1, 
        product_img2 = :product_img2, 
        product_img3 = :product_img3, 
        product_img4 = :product_img4, 
        product_img5 = :product_img5, 
        amount = :amount
        WHERE seller_id = :seller_id AND
        shop_id = :shop_id AND
         product_id = :product_id');
    
    // Bind Values
    $this->db->bind(':product_name', $data['product_name']);
    $this->db->bind(':product_desc', $data['product_desc']);
    $this->db->bind(':product_cat', $data['product_cat']);
    $this->db->bind(':product_img1', $data['product_img1']);
    $this->db->bind(':product_img2', $data['product_img2']);
    $this->db->bind(':product_img3', $data['product_img3']);
    $this->db->bind(':product_img4', $data['product_img4']);
    $this->db->bind(':product_img5', $data['product_img5']);
    $this->db->bind(':seller_id', $data['seller_id']);
    $this->db->bind(':shop_id', $data['shop_id']);
    $this->db->bind(':product_id', $data['product_id']);
    $this->db->bind(':amount', $data['amount']);
    
    // Execute
    if ($this->db->execute()) {
        return true;
    } else {
        return false;
    }
}


    // Add Post
    public function createProduct($data) {
      // Prepare Query
      $this->db->query('INSERT INTO products (
              product_name, 
              product_desc, 
              product_cat, 
              product_img1, 
              product_img2, 
              product_img3, 
              product_img4, 
              product_img5, 
              seller_id, 
              shop_id, 
              product_id,
              amount,
              uni
          ) VALUES (
              :product_name, 
              :product_desc,  
              :product_cat, 
              :product_img1, 
              :product_img2, 
              :product_img3, 
              :product_img4, 
              :product_img5, 
              :seller_id, 
              :shop_id, 
              :product_id,
              :amount,
              :uni
          )');
      
      // Bind Values
      $this->db->bind(':product_name', $data['product_name']);
      $this->db->bind(':product_desc', $data['product_desc']);
      $this->db->bind(':product_cat', $data['product_cat']);
      $this->db->bind(':product_img1', $data['product_img1']);
      $this->db->bind(':product_img2', $data['product_img2']);
      $this->db->bind(':product_img3', $data['product_img3']);
      $this->db->bind(':product_img4', $data['product_img4']);
      $this->db->bind(':product_img5', $data['product_img5']);
      $this->db->bind(':seller_id', $data['seller_id']);
      $this->db->bind(':shop_id', $data['shop_id']);
      $this->db->bind(':product_id', $data['product_id']);
      $this->db->bind(':amount', $data['amount']);
      $this->db->bind(':uni', $data['uni']);
      
      // Execute
      if($this->db->execute()){
          return true;
      } else {
          return false;
      }
  }
  
      public function postReview($da1, $da2, $da3, $da4, $da5, $da6)
   {
       echo $da1. " ".$da2. " ". $da3." ".$da4." ".$da5." ". $da6;
     try {
        $this->db->query('INSERT INTO reviewxx set product_id = :product_id, desc = :desc, rating = :rating, user_id = :user_id, username = :username, img = :img');

        $this->db->bind(':product_id', $da1);
        $this->db->bind(':desc', $da3);
        $this->db->bind(':rating', $da2);
        $this->db->bind(':user_id', $da4);
        $this->db->bind(':username', $da5);
        $this->db->bind(':img', $da6);

      

        if ($this->db->execute()) {
            return true;
        } else {
            return false; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }
    public function postReviewxx($data) {
          // Prepare Query
       $this->db->query('INSERT INTO reviewxx (
        product_id, 
        `desc`, 
        rating, 
        user_id, 
        username, 
        img
    ) VALUES (
        :product_id, 
        :desc,  
        :rating, 
        :user_id, 
        :username, 
        :img
    )');

      
      // Bind Values
      $this->db->bind(':product_id', $data['product_id']);
      $this->db->bind(':desc', $data['desc']);
      $this->db->bind(':rating', $data['rating']);
      $this->db->bind(':user_id', $data['user_id']);
      $this->db->bind(':username', $data['username']);
      $this->db->bind(':img', $data['img']);
  
      // Execute
      if($this->db->execute()){
          return true;
      } else {
          return false;
      }
  }
  
  
    public function createFoodProduct($data) {
      // Prepare Query
      $this->db->query('INSERT INTO food_products (
              product_name, 
              product_desc, 
              product_cat, 
              product_img1, 
              product_img2, 
              product_img3, 
              product_img4, 
              product_img5, 
              seller_id, 
              shop_id, 
              product_id,
              amount,
              uni
          ) VALUES (
              :product_name, 
              :product_desc, 
              :product_cat, 
              :product_img1, 
              :product_img2, 
              :product_img3, 
              :product_img4, 
              :product_img5, 
              :seller_id, 
              :shop_id, 
              :product_id,
              :amount,
              :uni
          )');
      
      // Bind Values
      $this->db->bind(':product_name', $data['product_name']);
      $this->db->bind(':product_desc', $data['product_desc']);
      $this->db->bind(':product_cat', $data['product_cat']);
      $this->db->bind(':product_img1', $data['product_img1']);
      $this->db->bind(':product_img2', $data['product_img2']);
      $this->db->bind(':product_img3', $data['product_img3']);
      $this->db->bind(':product_img4', $data['product_img4']);
      $this->db->bind(':product_img5', $data['product_img5']);
      $this->db->bind(':seller_id', $data['seller_id']);
      $this->db->bind(':shop_id', $data['shop_id']);
      $this->db->bind(':product_id', $data['product_id']);
      $this->db->bind(':amount', $data['amount']);
      $this->db->bind(':uni', $data['uni']);
      
      // Execute
      if($this->db->execute()){
          return true;
      } else {
          return false;
      }
  }
  
  
    public function addToWishList($data) {
      // Prepare Query
      $this->db->query('INSERT INTO product_wishlist (
              product_name, 
              product_desc, 
              product_cat, 
              product_img1, 
              product_img2, 
              product_img3, 
              product_img4, 
              product_img5, 
              seller_id,
              user_id,
              shop_id, 
              product_id,
              amount,
              uni
          ) VALUES (
              :product_name, 
              :product_desc,  
              :product_cat, 
              :product_img1, 
              :product_img2, 
              :product_img3, 
              :product_img4, 
              :product_img5, 
              :seller_id, 
              :user_id,
              :shop_id, 
              :product_id,
              :amount,
              :uni
          )');
      
      // Bind Values
      $this->db->bind(':product_name', $data['product_name']);
      $this->db->bind(':product_desc', $data['product_desc']);
      $this->db->bind(':product_cat', $data['product_cat']);
      $this->db->bind(':product_img1', $data['product_img1']);
      $this->db->bind(':product_img2', $data['product_img2']);
      $this->db->bind(':product_img3', $data['product_img3']);
      $this->db->bind(':product_img4', $data['product_img4']);
      $this->db->bind(':product_img5', $data['product_img5']);
      $this->db->bind(':seller_id', $data['seller_id']);
      $this->db->bind(':user_id', $data['user_id']);
      $this->db->bind(':shop_id', $data['shop_id']);
      $this->db->bind(':product_id', $data['product_id']);
      $this->db->bind(':amount', $data['amount']);
      $this->db->bind(':uni', $data['uni']);

      
      if($this->db->execute()){
          return true;
      } else {
          return false;
      }
      
    }
  
    public function removeFromWishList($Pid, $Uid) {
      // Prepare Query
      $this->db->query('DELETE FROM product_wishlist WHERE product_id = :product_id AND user_id = :user_id');
      
      $this->db->bind(':user_id', $Uid);
      $this->db->bind(':product_id', $Pid);
      
      
      
      if($this->db->execute()){
          return true;
      } else {
          return false;
      }
      
    }
  
    public function activateShop($data) {
        // Prepare Query
        $this->db->query('INSERT INTO shopdetails SET 
            shop_id = :shop_id, 
            shop_plan = :shop_plan, 
            plan_price = :plan_price, 
            expiry_date = :expiry_date, 
            seller_id = :seller_id');
        
        // Bind Values
        $this->db->bind(':shop_id', $data['shop_id']);
        $this->db->bind(':shop_plan', $data['shop_plan']);
        $this->db->bind(':plan_price', $data['plan_price']);
        $this->db->bind(':expiry_date', $data['expiry_date']);
        $this->db->bind(':seller_id', $data['seller_id']);
        
        // Execute
        if($this->db->execute()){
            $this->db->query('UPDATE initkey_rid SET 
            shop_id = :shop_id, 
            shop_plan = :shop_plan, 
            plan_price = :plan_price, 
            expiry_date = :expiry_date
            WHERE seller_id = :seller_id');
        
        // Bind Values
        $this->db->bind(':shop_id', $data['shop_id']);
        $this->db->bind(':shop_plan', $data['shop_plan']);
        $this->db->bind(':plan_price', $data['plan_price']);
        $this->db->bind(':expiry_date', $data['expiry_date']);
        $this->db->bind(':seller_id', $data['seller_id']);
        
        // Execute
        if($this->db->execute()){
            return true;
        } else {
            return false;
        }
        } else {
            return false;
        }
    }

 public function buyAirtime($data){
      // Prepare Query
      $this->db->query('INSERT INTO transactions (username, userid,reference, amount, status, email, tr_type) 
      VALUES (:username, :userid, :reference, :amount, :status, :email, :tr_type)');

      // Bind Values
      $this->db->bind(':username', $data['username']);
      $this->db->bind(':userid', $data['user_id']);
      $this->db->bind(':reference', $data['ref']);
      $this->db->bind(':amount', $data['amount']);
      $this->db->bind(':status', 'successful');
      $this->db->bind(':email', $data['email']);
    //   $this->db->bind(':network_id', $data['network']);
    //   $this->db->bind(':phone', $data['phone']);
      // $this->db->bind(':plan_id', $data['plan_id']);
      $this->db->bind(':tr_type', 'airtime');
      
      //Execute
      if($this->db->execute()){
       if ($this->reduceacc($data)) {
         return true;
       }else {
        return false;
       }
      } else {
        return false;
      }
    }

    // Update Post
    public function updatePost($data){
      // Prepare Query
      $this->db->query('UPDATE posts SET title = :title, body = :body WHERE id = :id');

      // Bind Values
      $this->db->bind(':id', $data['id']);
      $this->db->bind(':title', $data['title']);
      $this->db->bind(':body', $data['body']);
      
      //Execute
      if($this->db->execute()){
        return true;
      } else {
        return false;
      }
    }
    public function updateacc($data){
      // Prepare Query
      $this->db->query('SELECT * FROM user_account WHERE user_id = :user_id');
      $this->db->bind(':user_id', $data['user_id']);
      $row = $this->db->single();

      $this->db->query('UPDATE user_account SET acountbalance = :acountbalance WHERE user_id = :user_id');

      // Bind Values
      $this->db->bind(':user_id', $data['user_id']);
      $this->db->bind(':acountbalance', $data['amount'] + $row->acountbalance);
      
      
      //Execute
      if($this->db->execute()){
        return true;
      } else {
        return false;
      }
    }
    public function reduceacc($data){
      // Prepare Query
      $this->db->query('SELECT * FROM user_account WHERE user_id = :user_id');
      $this->db->bind(':user_id', $data['user_id']);
      $row = $this->db->single();

      $this->db->query('UPDATE user_account SET acountbalance = :acountbalance WHERE user_id = :user_id');

      // Bind Values
      $this->db->bind(':user_id', $data['user_id']);
      $this->db->bind(':acountbalance', $row->acountbalance - $data['amount']);


      $_SESSION['accb'] = $row->acountbalance - $data['amount'];
      
      
      //Execute
      if($this->db->execute()){
        return true;
      } else {
        return false;
      }
    }

    // Delete Post
    public function deleteProduct($id){
      // Prepare Query
      $this->db->query('DELETE FROM products WHERE id = :id');

      // Bind Values
      $this->db->bind(':id', $id);
      
      //Execute
      if($this->db->execute()){
        return true;
      } else {
        return false;
      }
    }
  }