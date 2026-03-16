<?php 
class Scan {
    private $db;

    public function __construct(){
        $this->db = new Database;
    }




    //Get all submule 
    public function getAppSettings(){
        $this->db->query("SELECT * FROM generalsetting	");
        $row = $this->db->single();
        // Check roow
        return $row;
         
      
    }
    

    public function save($data){
/*
INSERT INTO `generalsetting` (`id`, `siteName`, `state`, `city`, `country`, `webAdd`, `officeAdd`, `email`, `phone`, `pobox`, `zip`, `slogan`, `logo`, `activeCode`, `currentYear`, `jobSession`, `onOffEmail`, `onOffSms`, `onOffJobPosting`, `integrationKey`, `wallet1`, `wallet2`, `onOffMemberLogin`, `onOffMaintain`, `onOffShareAppLink`, `nisCost`, `surconCost`) VALUES ('6', '', '', '', '', '', '', '', '', '', '', '', '', '', '', '', '0', '', '0', '', '', '', '0', '0', '0', '', '');
*/

        $this->db->query('UPDATE  generalsetting  set siteName = :siteName,state = :state,city = :city,country = :country,webAdd = :webAdd,officeAdd = :officeAdd,email = :email,pobox = :pobox,zip = :zip,phone = :phone ,currentYear = :currentYear,jobSession = :jobSession,integrationKey = :integrationKey,wallet1 = :wallet1, slogan=:slogan ,wallet2 = :wallet2 ,nisCost = :nisCost ,surconCost = :surconCost ');
        // Bind Values
        
        $this->db->bind(':siteName', $data['siteName']);
        $this->db->bind(':state', $data['state']);
        $this->db->bind(':city', $data['city']);
        $this->db->bind(':country', $data['country']);
        $this->db->bind(':webAdd', $data['webAdd']);
        $this->db->bind(':officeAdd', $data['officeAdd']);
        $this->db->bind(':email', $data['email']);
        $this->db->bind(':phone', $data['phone']);
        $this->db->bind(':pobox', $data['pobox']);
        $this->db->bind('zip', $data['zip']);
        $this->db->bind(':slogan', $data['slogan']);
        $this->db->bind(':currentYear', $data['currentYear']);
        $this->db->bind(':jobSession', $data['jobSession']);
        $this->db->bind(':integrationKey', $data['integrationKey']);
        $this->db->bind(':wallet1', $data['wallet1']);
        $this->db->bind(':wallet2', $data['wallet2']);
        $this->db->bind(':nisCost', $data['nisCost']);
        $this->db->bind(':surconCost', $data['surconCost']);
         
         
        // Execute
        //$this->db->execute();


        if($this->db->execute()){
            return true;
        } else {
            return false;
        }

    }






 
}

